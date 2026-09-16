// Пакет model: типи домену v4 (схема — docs/DB_DESIGN_V4.md),
// розділені між store, scraper, api та фронтендом.
package model

import (
	"encoding/json"
	"time"
)

// Статуси замовлення (CHECK у БД). У v4 немає 'applied' — заявки прибрані.
const (
	StatusNew      = "new"
	StatusSeen     = "seen"
	StatusWon      = "won"
	StatusLost     = "lost"
	StatusArchived = "archived"
)

// Order — «листочок»: замовлення, зірване з джерела.
type Order struct {
	ID              int64      `json:"id"`
	SourceID        int32      `json:"source_id"`
	SourceKey       string     `json:"source_key"`        // JOIN із sources для UI
	SourceMessageID *string    `json:"source_message_id"` // дедуп-ключ
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Status          string     `json:"status"`
	BudgetCents     *int32     `json:"budget_cents"`
	ExternalURL     string     `json:"external_url"`
	PublishedAt     *time.Time `json:"published_at"`
	FirstSeenAt     time.Time  `json:"first_seen_at"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
	Skills          []string   `json:"skills"` // з order_skills (M:N)
}

// OrderFilter — критерії списку замовлень.
type OrderFilter struct {
	Status         string
	Skill          string // фільтр за навичкою (JOIN order_skills)
	MaxBudgetCents *int32
	DateFrom       *time.Time
	DateTo         *time.Time
	Query          string
	Limit          int
}

// Listing — сире замовлення з джерела до потрапляння в БД.
type Listing struct {
	SourceKey       string
	SourceMessageID string
	Title           string
	Description     string
	URL             string
	BudgetCents     *int32
	PublishedAt     *time.Time
	Skills          []string
}

// Profile — збережена стратегія пошуку (замість віток v2).
type Profile struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	MaxBudgetCents *int32     `json:"max_budget_cents"`
	DateFrom       *time.Time `json:"date_from"`
	DateTo         *time.Time `json:"date_to"`
	ResultsLimit    int32      `json:"results_limit"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	Skills         []string   `json:"skills"`     // з profile_skills (M:N)
	SourceIDs      []int32    `json:"source_ids"` // з profile_sources (M:N)
}

// Skill — навичка (довідник, поповнюється скрейпером).
type Skill struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Source — джерело збору.
type Source struct {
	ID      int32  `json:"id"`
	Key     string `json:"key"`
	Name    string `json:"name"`
	Kind    string `json:"kind"` // api|telegram|rss|html
	Enabled bool   `json:"enabled"`
}

// SourceChannel — окремий Telegram-канал усередині джерела.
type SourceChannel struct {
	ID       int32  `json:"id"`
	SourceID int32  `json:"source_id"`
	Handle   string `json:"handle"`
	Title    string `json:"title"`
	Enabled  bool   `json:"enabled"`
}

// ScrapeRun — лог виконання профілю.
type ScrapeRun struct {
	ID         int64      `json:"id"`
	ProfileID  int64      `json:"profile_id"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Outcome    string     `json:"outcome"` // ok|error
	ErrorText  string     `json:"error_text"`
	Stats      RunStats   `json:"stats"`
}

// RunStats — агрегати запуску (stats JSONB у scrape_runs).
type RunStats struct {
	Fetched int `json:"fetched"`
	Matched int `json:"matched"`
	New     int `json:"new"`
}

// ArchivedOrder — снапшот зниклого замовлення (archived_orders).
type ArchivedOrder struct {
	ID          int64           `json:"id"`
	SourceKey   string          `json:"source_key"`
	Title       string          `json:"title"`
	Status      string          `json:"status"`
	BudgetCents *int32          `json:"budget_cents"`
	ExternalURL string          `json:"external_url"`
	Snapshot    json.RawMessage `json:"snapshot"`
	Reason      string          `json:"reason"` // removed|manual
	ArchivedAt  time.Time       `json:"archived_at"`
}
