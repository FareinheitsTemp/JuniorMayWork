// Пакет model: типи домену v4 (схема — docs/DB_DESIGN_V4.md),
// розділені між store, scraper, api та фронтендом.
package model

import (
	"encoding/json"
	"time"
)

const (
	StatusNew      = "new"
	StatusSeen     = "seen"
	StatusWon      = "won"
	StatusLost     = "lost"
	StatusArchived = "archived"
)

type Order struct {
	ID              int64      `json:"id"`
	SourceID        int32      `json:"source_id"`
	SourceKey       string     `json:"source_key"`
	SourceMessageID *string    `json:"source_message_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Status          string     `json:"status"`
	BudgetCents     *int32     `json:"budget_cents"`
	ExternalURL     string     `json:"external_url"`
	PublishedAt     *time.Time `json:"published_at"`
	FirstSeenAt     time.Time  `json:"first_seen_at"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
	Skills          []string   `json:"skills"`
}

type OrderPatch struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	BudgetCents *int32  `json:"budget_cents"`
}

type OrderFilter struct {
	Status         string
	Skill          string
	MaxBudgetCents *int32
	DateFrom       *time.Time
	DateTo         *time.Time
	Query          string
	Limit          int
}

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

type Profile struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	MaxBudgetCents *int32     `json:"max_budget_cents"`
	DateFrom       *time.Time `json:"date_from"`
	DateTo         *time.Time `json:"date_to"`
	ResultsLimit   int32      `json:"results_limit"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	Skills         []string   `json:"skills"`
	SourceIDs      []int32    `json:"source_ids"`
}

type Skill struct { ID int64 `json:"id"`; Name string `json:"name"` }
type Source struct { ID int32 `json:"id"`; Key string `json:"key"`; Name string `json:"name"`; Kind string `json:"kind"`; Enabled bool `json:"enabled"` }
type SourceChannel struct { ID int32 `json:"id"`; SourceID int32 `json:"source_id"`; Handle string `json:"handle"`; Title string `json:"title"`; Enabled bool `json:"enabled"` }

type ScrapeRun struct {
	ID int64 `json:"id"`
	ProfileID int64 `json:"profile_id"`
	StartedAt time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Outcome string `json:"outcome"`
	ErrorText string `json:"error_text"`
	Stats RunStats `json:"stats"`
}
type RunStats struct { Fetched int `json:"fetched"`; Matched int `json:"matched"`; New int `json:"new"` }
type ArchivedOrder struct {
	ID int64 `json:"id"`; SourceKey string `json:"source_key"`; Title string `json:"title"`; Status string `json:"status"`
	BudgetCents *int32 `json:"budget_cents"`; ExternalURL string `json:"external_url"`; Snapshot json.RawMessage `json:"snapshot"`
	Reason string `json:"reason"`; ArchivedAt time.Time `json:"archived_at"`
}
