// Пакет model: типи домену, що їх розділяють store, scraper, api і фронтенд.
package model

import (
	"encoding/json"
	"time"
)

// Branch — "вітка" дерева: нішa/МП з ключовими словами і лімітом бюджету.
type Branch struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Keywords       []string  `json:"keywords"`
	MaxBudgetCents int       `json:"max_budget_cents"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

// Order — "листочок": замовлення, зірване з джерела.
type Order struct {
	ID           int64           `json:"id"`
	Source       string          `json:"source"`
	ExternalID   string          `json:"external_id"`
	URL          string          `json:"url"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	BudgetCents  *int            `json:"budget_cents"`
	Currency     string          `json:"currency"`
	Skills       []string        `json:"skills"`
	BranchID     *int64          `json:"branch_id"`
	Status       string          `json:"status"`
	FirstSeenAt  time.Time       `json:"first_seen_at"`
	LastSeenAt   time.Time       `json:"last_seen_at"`
	Raw          json.RawMessage `json:"raw"`
}

// Event — запис живої історії: нове/оновлене/видалене замовлення, зміна статусу, заявка.
type Event struct {
	ID        int64           `json:"id"`
	OrderID   *int64          `json:"order_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// Application — заявка на замовлення та її результат.
type Application struct {
	ID         int64     `json:"id"`
	OrderID    *int64    `json:"order_id"`
	OrderTitle string    `json:"order_title"`
	Note       string    `json:"note"`
	Result     string    `json:"result"`
	AppliedAt  time.Time `json:"applied_at"`
}

// Listing — сире замовлення з джерела до потрапляння в БД.
type Listing struct {
	Source      string
	ExternalID  string
	URL         string
	Title       string
	Description string
	BudgetCents *int
	Currency    string
	Skills      []string
	Raw         json.RawMessage
}

// EventPayload — снапшот у events.payload: живе навіть після видалення замовлення.
type EventPayload struct {
	OrderID int64  `json:"order_id"`
	Source  string `json:"source"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Status  string `json:"status,omitempty"`
}

func (p EventPayload) JSON() json.RawMessage {
	b, _ := json.Marshal(p)
	return b
}
