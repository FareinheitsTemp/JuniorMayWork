package model

import "time"

// SearchProfile is a persistent, user-defined collection strategy.
// Tags and Sources are loaded with the profile for direct API use.
type SearchProfile struct {
	ID              int64                 `json:"id"`
	Name            string                `json:"name"`
	IsActive        bool                  `json:"is_active"`
	IntervalMinutes int                   `json:"interval_minutes"`
	MinBudgetCents  *int                  `json:"min_budget_cents,omitempty"`
	MaxBudgetCents  *int                  `json:"max_budget_cents,omitempty"`
	MaxAgeHours     *int                  `json:"max_age_hours,omitempty"`
	Tags            []SearchProfileTag    `json:"tags"`
	Sources         []SearchProfileSource `json:"sources"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type SearchProfileTag struct {
	ID        int64     `json:"id"`
	ProfileID int64     `json:"profile_id"`
	Tag       string    `json:"tag"`
	TagType   string    `json:"tag_type"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchProfileSource struct {
	ProfileID int64 `json:"profile_id"`
	SourceID  int64 `json:"source_id"`
	IsEnabled bool  `json:"is_enabled"`
}

type SearchRun struct {
	ID              int64      `json:"id"`
	ProfileID       int64      `json:"profile_id"`
	Trigger         string     `json:"trigger"`
	Status          string     `json:"status"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	DurationMS      *int64     `json:"duration_ms,omitempty"`
	OrdersFound     int        `json:"orders_found"`
	OrdersNew       int        `json:"orders_new"`
	DuplicatesCount int        `json:"duplicates_count"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type OrderDiscovery struct {
	ID              int64      `json:"id"`
	OrderID         int64      `json:"order_id"`
	SearchRunID     int64      `json:"search_run_id"`
	SourceID        int64      `json:"source_id"`
	SourceChannelID *int64     `json:"source_channel_id,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CapturedAt      time.Time  `json:"captured_at"`
	RelevanceScore  float64    `json:"relevance_score"`
	MatchedTags     []string   `json:"matched_tags"`
	IsNewOrder      bool       `json:"is_new_order"`
}

type ReportExport struct {
	ID          int64     `json:"id"`
	SearchRunID int64     `json:"search_run_id"`
	FileName    string    `json:"file_name"`
	StorageKey  string    `json:"storage_key"`
	MimeType    string    `json:"mime_type"`
	OrdersTotal int       `json:"orders_total"`
	CreatedAt   time.Time `json:"created_at"`
}
