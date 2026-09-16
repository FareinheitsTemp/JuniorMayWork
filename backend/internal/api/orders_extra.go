package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

// Orders extra — HTTP-хендлери для нотаток, історії статусів і зміни статусу.
// Реєстрація в cmd/jmw/extra_routes.go:
//
//	extra.HandleFunc("GET /api/orders/statuses", server.HandleOrderStatuses)
//	extra.HandleFunc("GET /api/orders/{id}/notes", server.HandleOrderNotes)
//	extra.HandleFunc("POST /api/orders/{id}/notes", server.HandleOrderNoteCreate)
//	extra.HandleFunc("GET /api/orders/{id}/history", server.HandleOrderHistory)
//	extra.HandleFunc("PATCH /api/orders/{id}/status", server.HandleOrderStatusUpdate)

func orderIDFromPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "неправильний id замовлення")
		return 0, false
	}
	return id, true
}

func (s *Server) HandleOrderStatuses(w http.ResponseWriter, r *http.Request) {
	statuses, err := s.store.ListOrderStatuses(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"statuses": statuses})
}

func (s *Server) HandleOrderNotes(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	notes, err := s.store.ListOrderNotes(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
}

func (s *Server) HandleOrderNoteCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний JSON")
		return
	}
	note, err := s.store.CreateOrderNote(r.Context(), id, body.Body)
	if err != nil {
		if errors.Is(err, store.ErrEmptyNote) {
			writeErr(w, http.StatusBadRequest, "текст нотатки порожній")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "замовлення не знайдено")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (s *Server) HandleOrderHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	changes, err := s.store.ListOrderStatusHistory(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": changes})
}

func (s *Server) HandleOrderStatusUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	id, ok := orderIDFromPath(w, r)
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
		Actor  string `json:"actor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний JSON")
		return
	}
	if body.Status == "" {
		writeErr(w, http.StatusBadRequest, "не вказано статус")
		return
	}
	status, err := s.store.ChangeOrderStatus(r.Context(), id, body.Status, body.Actor)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "замовлення або статус не знайдено")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}
