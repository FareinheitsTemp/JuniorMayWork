// Пакет scraper: збір замовлень із зовнішніх джерел для схеми v4.
package scraper

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/config"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/scraper/sources"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

// Manager координує опитування всіх увімкнених джерел.
type Manager struct {
	store *store.Store
	cfg   config.Config
	log   *slog.Logger

	mu      sync.Mutex
	lastRun time.Time
	running bool
}

func NewManager(st *store.Store, cfg config.Config, log *slog.Logger) *Manager {
	return &Manager{store: st, cfg: cfg, log: log}
}

// RunOnce збирає замовлення з кожного увімкненого відомого джерела.
// Одночасний запуск блокується, щоб не створювати дублікати source_runs.
func (m *Manager) RunOnce(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		m.log.Debug("скрейпер уже запущений")
		return
	}
	m.running = true
	m.lastRun = time.Now()
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	available := sources.Sources()
	enabled, err := m.store.ListSources(ctx, true)
	if err != nil {
		m.log.Error("не вдалося прочитати sources", "err", err)
		return
	}

	for _, source := range enabled {
		fetch, ok := available[source.Key]
		if !ok {
			m.log.Warn("для джерела немає fetcher-а", "source", source.Key)
			continue
		}
		m.collectSource(ctx, source, fetch)
	}
}

// Start запускає фонове опитування; перший цикл виконується одразу.
func (m *Manager) Start(ctx context.Context) {
	m.RunOnce(ctx)
	interval := m.cfg.PollInterval
	if interval <= 0 {
		interval = 90 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.RunOnce(ctx)
		}
	}
}

func (m *Manager) collectSource(ctx context.Context, source model.Source, fetch sources.FetchFunc) {
	run, err := m.store.StartSourceRun(ctx, source.ID)
	if err != nil {
		m.log.Error("не вдалося створити source run", "source", source.Key, "err", err)
		return
	}

	listings, err := fetch(ctx)
	if err != nil {
		m.log.Warn("не вдалося прочитати джерело", "source", source.Key, "err", err)
		if finishErr := m.store.FinishSourceRun(ctx, run.ID, "error", err.Error(), model.RunStats{}); finishErr != nil {
			m.log.Error("не вдалося завершити source run", "id", run.ID, "err", finishErr)
		}
		return
	}

	stats := model.RunStats{Fetched: len(listings)}
	for _, listing := range listings {
		listing.SourceKey = source.Key
		order, inserted, upsertErr := m.store.UpsertListing(ctx, listing)
		if upsertErr != nil {
			m.log.Warn("не вдалося зберегти замовлення", "source", source.Key, "err", upsertErr)
			continue
		}
		if inserted {
			stats.New++
		}
		if m.matchesActiveProfile(ctx, order) {
			stats.Matched++
		}
	}

	staleBefore := time.Now().Add(-2 * m.cfg.PollInterval)
	stale, staleErr := m.store.StaleOrders(ctx, source.Key, staleBefore)
	if staleErr != nil {
		m.log.Warn("не вдалося знайти застарілі замовлення", "source", source.Key, "err", staleErr)
	} else if len(stale) > 0 {
		ids := make([]int64, 0, len(stale))
		for _, order := range stale { ids = append(ids, order.ID) }
		count, archiveErr := m.store.ArchiveOrders(ctx, ids, "removed")
		if archiveErr != nil {
			m.log.Warn("не вдалося заархівувати замовлення", "source", source.Key, "err", archiveErr)
		} else {
			m.log.Info("заархівовано зниклі замовлення", "source", source.Key, "count", count)
		}
	}

	if err := m.store.FinishSourceRun(ctx, run.ID, "ok", "", stats); err != nil {
		m.log.Error("не вдалося завершити source run", "id", run.ID, "err", err)
	}
}

// matchesActiveProfile перевіряє лише факт збігу з активним профілем.
// Детальні order_matches записуватиме наступний етап запуску профілів.
func (m *Manager) matchesActiveProfile(ctx context.Context, order model.Order) bool {
	profiles, err := m.store.ListProfiles(ctx)
	if err != nil { return false }
	for _, profile := range profiles {
		if !profile.IsActive { continue }
		if profile.MaxBudgetCents != nil && (order.BudgetCents == nil || *order.BudgetCents > *profile.MaxBudgetCents) { continue }
		if profile.DateFrom != nil && (order.PublishedAt == nil || order.PublishedAt.Before(*profile.DateFrom)) { continue }
		if profile.DateTo != nil && (order.PublishedAt == nil || order.PublishedAt.After(*profile.DateTo)) { continue }
		if len(profile.SourceIDs) > 0 && !containsSource(profile.SourceIDs, order.SourceID) { continue }
		if len(profile.Skills) > 0 && !hasSharedSkill(profile.Skills, order.Skills) { continue }
		return true
	}
	return false
}

func containsSource(ids []int32, id int32) bool {
	for _, item := range ids { if item == id { return true } }
	return false
}

func hasSharedSkill(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, item := range a { set[item] = struct{}{} }
	for _, item := range b { if _, ok := set[item]; ok { return true } }
	return false
}

func (m *Manager) LastRun() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastRun
}

func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}
