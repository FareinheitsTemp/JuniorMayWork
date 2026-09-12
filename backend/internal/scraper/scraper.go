// Пакет scraper: менеджер збору замовлень — poll, diff, класифікація, архівація.
package scraper

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/archive"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/config"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/scraper/sources"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

// Notifier отримує події для розсилки в реальному часі (WebSocket hub).
type Notifier interface {
	Notify(event model.Event)
}

type Status struct {
	Running  bool      `json:"running"`
	LastRun  time.Time `json:"last_run"`
	NextRun  time.Time `json:"next_run"`
	Interval string    `json:"interval"`
	Sources  []string  `json:"sources"`
}

type Manager struct {
	store   *store.Store
	archive *archive.Archiver
	notify  Notifier
	cfg     config.Config
	log     *slog.Logger
	running atomic.Bool
	mu      sync.Mutex
	lastRun time.Time
}

func NewManager(st *store.Store, ar *archive.Archiver, n Notifier, cfg config.Config, log *slog.Logger) *Manager {
	return &Manager{store: st, archive: ar, notify: n, cfg: cfg, log: log}
}

// Run: перший збір одразу, далі — за інтервалом, до скасування контексту.
func (m *Manager) Run(ctx context.Context) {
	m.collect(ctx)
	ticker := time.NewTicker(m.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collect(ctx)
		}
	}
}

// RunOnce — ручний запуск (POST /api/scraper/run), асинхронний.
func (m *Manager) RunOnce(ctx context.Context) {
	go m.collect(ctx)
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Status{
		Running:  m.running.Load(),
		LastRun:  m.lastRun,
		NextRun:  m.lastRun.Add(m.cfg.PollInterval),
		Interval: m.cfg.PollInterval.String(),
		Sources:  m.cfg.Sources,
	}
}

func (m *Manager) collect(ctx context.Context) {
	if !m.running.CompareAndSwap(false, true) {
		return
	}
	defer m.running.Store(false)

	fetchers := sources.Sources()
	for _, name := range m.cfg.Sources {
		fetch, ok := fetchers[name]
		if !ok {
			m.log.Warn("невідоме джерело, пропускаю", "source", name)
			continue
		}
		listings, err := fetch(ctx)
		if err != nil {
			// помилка мережі не тригерить архівацію — stale-поріг дає запас часу
			m.log.Error("джерело недоступне", "source", name, "err", err)
			continue
		}
		m.ingest(ctx, name, listings)
	}

	m.mu.Lock()
	m.lastRun = time.Now()
	m.mu.Unlock()
}

func (m *Manager) ingest(ctx context.Context, sourceName string, listings []model.Listing) {
	branches, err := m.store.ListBranches(ctx, true)
	if err != nil {
		m.log.Error("не вдалося прочитати вітки", "err", err)
	}

	for _, l := range listings {
		l.Source = sourceName
		order, inserted, err := m.store.UpsertListing(ctx, l)
		if err != nil {
			m.log.Error("upsert замовлення", "source", sourceName, "err", err)
			continue
		}
		if !inserted {
			continue // оновлення вже відомого листочка — тихо
		}
		if branchID := classify(order, branches); branchID != nil {
			if err := m.store.SetOrderBranch(ctx, order.ID, branchID); err != nil {
				m.log.Error("класифікація замовлення", "id", order.ID, "err", err)
			}
		}
		m.emit(ctx, &order.ID, "new", order, order.Status)
	}

	// зниклі: не було в БД останні 2 вибірки — в архів і з БД геть.
	staleBefore := time.Now().Add(-2 * m.cfg.PollInterval)
	stale, err := m.store.StaleOrders(ctx, sourceName, staleBefore)
	if err != nil {
		m.log.Error("пошук зниклих замовлень", "source", sourceName, "err", err)
		return
	}
	for _, o := range stale {
		if err := m.archive.Append(o); err != nil {
			m.log.Error("архівація замовлення", "id", o.ID, "err", err)
			continue
		}
		m.emit(ctx, &o.ID, "removed", o, o.Status)
		if err := m.store.DeleteOrder(ctx, o.ID); err != nil {
			m.log.Error("видалення замовлення з БД", "id", o.ID, "err", err)
		}
	}
}

// emit: подія в БД + розсилка підписникам WebSocket.
func (m *Manager) emit(ctx context.Context, orderID *int64, typ string, o model.Order, status string) {
	payload := model.EventPayload{
		OrderID:     o.ID,
		Source:      o.Source,
		URL:         o.URL,
		Title:       o.Title,
		Status:      status,
		BudgetCents: o.BudgetCents,
	}
	if err := m.store.InsertEvent(ctx, orderID, typ, payload); err != nil {
		m.log.Error("запис події", "type", typ, "err", err)
	}
	m.notify.Notify(model.Event{OrderID: orderID, Type: typ, Payload: payload.JSON(), CreatedAt: time.Now()})
}

// classify підбирає вітку з найбільшим збігом ключових слів
// (бюджет замовлення, якщо відомий, має вкладатися в ліміт вітки).
func classify(o model.Order, branches []model.Branch) *int64 {
	if len(branches) == 0 {
		return nil
	}
	text := strings.ToLower(o.Title + "\n" + o.Description)
	var best *int64
	bestScore := 0
	for _, b := range branches {
		if o.BudgetCents != nil && *o.BudgetCents > b.MaxBudgetCents {
			continue
		}
		score := 0
		for _, kw := range b.Keywords {
			kw = strings.ToLower(strings.TrimSpace(kw))
			if kw != "" && strings.Contains(text, kw) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			id := b.ID
			best = &id
		}
	}
	return best
}
