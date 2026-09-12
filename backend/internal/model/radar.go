package model

import (
	"encoding/json"
	"time"
)

// Source — контрольований канал збору: API, Telegram, RSS або публічний HTML.
type Source struct {
	ID                  int64      `json:"id"`
	Key                 string     `json:"key"`
	Name                string     `json:"name"`
	Kind                string     `json:"kind"`
	BaseURL             string     `json:"base_url"`
	Enabled             bool       `json:"enabled"`
	PollIntervalSeconds int        `json:"poll_interval_seconds"`
	LastSuccessAt       *time.Time `json:"last_success_at"`
	LastFailureAt       *time.Time `json:"last_failure_at"`
	LastError           string     `json:"last_error"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// SourceChannel — окремий Telegram-канал, що належить source=telegram.
type SourceChannel struct {
	ID        int64     `json:"id"`
	SourceID  int64     `json:"source_id"`
	Handle    string    `json:"handle"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// SourceRun — результат однієї спроби збору, основа source health і pipeline.
type SourceRun struct {
	ID              int64           `json:"id"`
	SourceID        int64           `json:"source_id"`
	StartedAt       time.Time       `json:"started_at"`
	FinishedAt      *time.Time      `json:"finished_at"`
	Outcome         string          `json:"outcome"`
	DiscoveredCount int             `json:"discovered_count"`
	InsertedCount   int             `json:"inserted_count"`
	UpdatedCount    int             `json:"updated_count"`
	DuplicateCount  int             `json:"duplicate_count"`
	HTTPStatus      *int            `json:"http_status"`
	ErrorMessage    string          `json:"error_message"`
	Meta            json.RawMessage `json:"meta"`
}

// SchemaLayout — збережена камера/масштаб ERD-мапи.
type SchemaLayout struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Viewport  json.RawMessage `json:"viewport"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// SchemaNode — позиція таблиці на інтерактивній мапі БД.
type SchemaNode struct {
	ID        int64   `json:"id"`
	LayoutID  int64   `json:"layout_id"`
	TableKey  string  `json:"table_key"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Color     string  `json:"color"`
	Collapsed bool    `json:"collapsed"`
}

// FreshOrder — order з алгоритмічним пріоритетом Radar.
type FreshOrder struct {
	Order
	SourcePublishedAt *time.Time `json:"source_published_at"`
	FreshnessScore    float64    `json:"freshness_score"`
	RelevanceScore    float64    `json:"relevance_score"`
	PriorityScore     float64    `json:"priority_score"`
	IsSeenByUser      bool       `json:"is_seen_by_user"`
	IsDismissed       bool       `json:"is_dismissed"`
	DuplicateOfID     *int64     `json:"duplicate_of_id"`
}
