// Пакет api: замовлення — список із фільтрами, редагування, видалення, заявка.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

func (s *Server) HandleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	q := r.URL.Query()
	f := store.OrderFilter{Status: q.Get("status"), Query: q.Get("q")}
	if v := q.Get("branch_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "некоректний branch_id")
			return
		}
		f.BranchID = &id
	}
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	list, err := s.store.ListOrders(r.Context(), f)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type orderPatch struct {
	Status      *string  `json:"status"`
	BranchID    *int64   `json:"branch_id"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	URL         *string  `json:"url"`
	BudgetCents *int     `json:"budget_cents"`
	Skills      []string `json:"skills"`
}

func (s *Server) HandleOrder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		o, err := s.store.GetOrder(r.Context(), id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, o)
	case http.MethodPatch:
		var p orderPatch
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeErr(w, http.StatusBadRequest, "некоректний JSON")
			return
		}
		if p.Status != nil && !orderStatuses[*p.Status] {
			writeErr(w, http.StatusBadRequest, "невідомий статус")
			return
		}
		current, err := s.store.GetOrder(r.Context(), id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		oldStatus := current.Status
		if p.Status != nil {
			current.Status = *p.Status
		}
		if p.BranchID != nil {
			current.BranchID = p.BranchID
		}
		if p.Title != nil {
			current.Title = *p.Title
		}
		if p.Description != nil {
			current.Description = *p.Description
		}
		if p.URL != nil {
			current.URL = *p.URL
		}
		if p.BudgetCents != nil {
			current.BudgetCents = p.BudgetCents
		}
		if p.Skills != nil {
			current.Skills = p.Skills
		}
		updated, err := s.store.UpdateOrder(r.Context(), id, current)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if p.Status != nil && *p.Status != oldStatus {
			s.emitStatusChanged(r, id, updated)
		}
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := s.store.DeleteOrder(r.Context(), id); err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "видалено"})
	default:
		w.Header().Set("Allow", "GET, PATCH, DELETE")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
	}
}

func (s *Server) emitStatusChanged(r *http.Request, id int64, o model.Order) {
	payload := model.EventPayload{
		OrderID: id, Source: o.Source, URL: o.URL, Title: o.Title, Status: o.Status,
	}
	if err := s.store.InsertEvent(r.Context(), &id, "status_changed", payload); err != nil {
		s.log.Warn("запис події status_changed", "id", id, "err", err)
	}
	s.notify.Notify(model.Event{
		OrderID:   &id,
		Type:      "status_changed",
		Payload:   payload.JSON(),
		CreatedAt: time.Now(),
	})
}

// HandleApply — заявка на замовлення: POST /api/orders/{id}/apply.
func (s *Server) HandleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний id")
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	app, err := s.store.ApplyToOrder(r.Context(), id, body.Note)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if o, err := s.store.GetOrder(r.Context(), id); err == nil {
		s.notify.Notify(model.Event{
			OrderID:   &id,
			Type:      "applied",
			Payload: model.EventPayload{
				OrderID: id, Source: o.Source, URL: o.URL, Title: o.Title, Status: o.Status,
			}.JSON(),
			CreatedAt: time.Now(),
		})
	}
	writeJSON(w, http.StatusCreated, app)
}
