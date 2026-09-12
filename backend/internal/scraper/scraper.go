// Пакет scraper: Radar — adaptive poll, source telemetry, dedup, freshness і priority.
package scraper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"math"
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

// Run: перший збір одразу; далі перевіряє джерела за глобальним коротким tick,
// але кожне source має власний інтервал і failure-backoff до 30 хвилин.
func (m *Manager) Run(ctx context.Context) {
	m.collect(ctx)
	ticker := time.NewTicker(30 * time.Second)
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
		Interval: "adaptive (30s–30m)",
		Sources:  m.cfg.Sources,
	}
}

func (m *Manager) collect(ctx context.Context) {
	if !m.running.CompareAndSwap(false, true) {
		return
	}
	defer m.running.Store(false)

	registered, err := m.store.ListSources(ctx)
	if err != nil {
		m.log.Error("не вдалося прочитати Radar sources", "err", err)
		return
	}
	fetchers := sources.Sources()
	now := time.Now()
	for _, source := range registered {
		if !source.Enabled || !sourceDue(source, now) {
			continue
		}
		fetch, ok := fetchers[source.Key]
		if !ok {
			m.log.Warn("джерело не має fetcher, пропускаю", "source", source.Key)
			continue
		}
		m.collectSource(ctx, source, fetch)
	}

	m.mu.Lock()
	m.lastRun = now
	m.mu.Unlock()
}

func sourceDue(source model.Source, now time.Time) bool {
	interval := time.Duration(source.PollIntervalSeconds) * time.Second
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}
	if source.LastFailureAt != nil && (source.LastSuccessAt == nil || source.LastFailureAt.After(*source.LastSuccessAt)) {
		// Error не означає «ддось ще раз зараз»: backoff до 30 хв.
		interval = minDuration(interval*3, 30*time.Minute)
		return now.Sub(*source.LastFailureAt) >= interval
	}
	if source.LastSuccessAt == nil {
		return true
	}
	return now.Sub(*source.LastSuccessAt) >= interval
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func (m *Manager) collectSource(ctx context.Context, source model.Source, fetch sources.FetchFunc) {
	run, err := m.store.CreateSourceRun(ctx, source.ID)
	if err != nil {
		m.log.Error("не вдалося створити source run", "source", source.Key, "err", err)
		return
	}

	listings, fetchErr := fetch(ctx)
	if fetchErr != nil {
		run.Outcome = "failure"
		run.ErrorMessage = fetchErr.Error()
		if _, err := m.store.FinishSourceRun(ctx, run); err != nil {
			m.log.Error("не вдалося завершити source run", "err", err)
		}
		m.log.Warn("джерело недоступне; увімкнено backoff", "source", source.Key, "err", fetchErr)
		return
	}

	run.DiscoveredCount = len(listings)
	branches, err := m.store.ListBranches(ctx, true)
	if err != nil {
		m.log.Error("не вдалося прочитати вітки", "err", err)
	}

	for _, listing := range listings {
		listing.Source = source.Key
		fingerprint := listingFingerprint(listing)
		duplicateOf, err := m.store.FindDuplicateByFingerprint(ctx, fingerprint)
		if err != nil {
			m.log.Warn("пошук дубліката", "source", source.Key, "err", err)
		}

		order, inserted, err := m.store.UpsertListing(ctx, listing)
		if err != nil {
			m.log.Error("upsert замовлення", "source", source.Key, "err", err)
			continue
		}
		if duplicateOf != nil && *duplicateOf != order.ID {
			run.DuplicateCount++
		}

		branchID, relevance := classify(order, branches)
		if branchID != nil {
			if err := m.store.SetOrderBranch(ctx, order.ID, branchID); err != nil {
				m.log.Error("класифікація замовлення", "id", order.ID, "err", err)
			}
		}
		freshness := freshnessScore(listing.PublishedAt, order.FirstSeenAt)
		priority := 0.55*freshness + 0.35*relevance + 10 // 10 = healthy source baseline
		if err := m.store.UpdateOrderRadar(ctx, order.ID, listing.PublishedAt, fingerprint, freshness, relevance, priority, duplicateOf); err != nil {
			m.log.Error("збереження radar score", "id", order.ID, "err", err)
		}

		if inserted {
			run.InsertedCount++
			// Лише унікальний новий гіг стає видимою live-подією.
			if duplicateOf == nil || *duplicateOf == order.ID {
				m.emit(ctx, &order.ID, "new", order, order.Status)
			}
		} else {
			run.UpdatedCount++
		}
	}

	if run.DiscoveredCount == 0 {
		run.Outcome = "warning"
		run.ErrorMessage = "джерело відповіло, але не повернуло замовлень"
	} else {
		run.Outcome = "success"
	}
	if _, err := m.store.FinishSourceRun(ctx, run); err != nil {
		m.log.Error("не вдалося завершити source run", "err", err)
	}

	m.archiveStale(ctx, source.Key)
}

func (m *Manager) archiveStale(ctx context.Context, sourceKey string) {
	// Зниклі: не було в БД 2×глобального інтервалу — в архів і з БД геть.
	staleBefore := time.Now().Add(-2 * m.cfg.PollInterval)
	stale, err := m.store.StaleOrders(ctx, sourceKey, staleBefore)
	if err != nil {
		m.log.Error("пошук зниклих замовлень", "source", sourceKey, "err", err)
		return
	}
	for _, order := range stale {
		if err := m.archive.Append(order); err != nil {
			m.log.Error("архівація замовлення", "id", order.ID, "err", err)
			continue
		}
		m.emit(ctx, &order.ID, "removed", order, order.Status)
		if err := m.store.DeleteOrder(ctx, order.ID); err != nil {
			m.log.Error("видалення замовлення з БД", "id", order.ID, "err", err)
		}
	}
}

// SHA-256 від нормалізованого title+description — крос-джерельна дедуплікація.
func listingFingerprint(listing model.Listing) string {
	text := strings.ToLower(listing.Title + " " + listing.Description)
	text = strings.Join(strings.Fields(text), " ")
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", sum)
}

// Свіжість згасає експоненційно: 100 зараз, ~37 через 24 години, ~5 через 3 дні.
func freshnessScore(publishedAt *time.Time, firstSeenAt time.Time) float64 {
	at := firstSeenAt
	if publishedAt != nil {
		at = *publishedAt
	}
	hours := math.Max(0, time.Since(at).Hours())
	return math.Round(100*math.Exp(-hours/24)*100) / 100
}

// relevance: ключові слова вітки + базові ключі веб-розробки. Повертає кращу вітку.
func classify(order model.Order, branches []model.Branch) (*int64, float64) {
	text := strings.ToLower(order.Title + "\n" + order.Description)
	var best *int64
	bestScore := 0.0
	for _, branch := range branches {
		if order.BudgetCents != nil && *order.BudgetCents > branch.MaxBudgetCents {
			continue
		}
		hits := 0
		for _, keyword := range branch.Keywords {
			keyword = strings.ToLower(strings.TrimSpace(keyword))
			if keyword != "" && strings.Contains(text, keyword) {
				hits++
			}
		}
		score := float64(hits) * 18
		if score > bestScore {
			bestScore = score
			id := branch.ID
			best = &id
		}
	}
	for _, keyword := range []string{"react", "javascript", "next.js", "typescript", "frontend", "лендінг", "верстка"} {
		if strings.Contains(text, keyword) {
			bestScore += 4
		}
	}
	if bestScore > 100 {
		bestScore = 100
	}
	return best, bestScore
}

// emit: подія в БД + розсилка підписникам WebSocket.
func (m *Manager) emit(ctx context.Context, orderID *int64, typ string, order model.Order, status string) {
	payload := model.EventPayload{
		OrderID:     order.ID,
		Source:      order.Source,
		URL:         order.URL,
		Title:       order.Title,
		Status:      status,
		BudgetCents: order.BudgetCents,
	}
	if err := m.store.InsertEvent(ctx, orderID, typ, payload); err != nil {
		m.log.Error("запис події", "type", typ, "err", err)
	}
	m.notify.Notify(model.Event{OrderID: orderID, Type: typ, Payload: payload.JSON(), CreatedAt: time.Now()})
}
