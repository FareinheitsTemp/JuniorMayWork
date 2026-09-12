// Пакет api: Fresh Radar, pipeline source runs і інтерактивна схема БД.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

func (s *Server) HandleFreshOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	limit := 8
	minPriority := 0.0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := r.URL.Query().Get("min_priority"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			minPriority = n
		}
	}
	orders, err := s.store.FreshOrders(r.Context(), limit, minPriority)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) HandleOrderSeen(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.MarkOrderSeen(r.Context(), id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "позначено як побачене"})
}

func (s *Server) HandleOrderDismiss(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.DismissOrder(r.Context(), id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "відхилено"})
}

func (s *Server) HandleRadarRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	var sourceID *int64
	if v := r.URL.Query().Get("source_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "некоректний source_id")
			return
		}
		sourceID = &id
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	runs, err := s.store.ListSourceRuns(r.Context(), sourceID, limit)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Server) HandleSchemaLayout(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Основна карта"
	}
	switch r.Method {
	case http.MethodGet:
		layout, nodes, err := s.store.GetSchemaLayout(r.Context(), name)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"layout": layout, "nodes": nodes})
	case http.MethodPatch:
		var body struct {
			ID       int64           `json:"id"`
			Viewport json.RawMessage `json:"viewport"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "некоректний JSON")
			return
		}
		if body.ID == 0 {
			writeErr(w, http.StatusBadRequest, "id обов'язковий")
			return
		}
		layout, err := s.store.UpdateSchemaViewport(r.Context(), body.ID, body.Viewport)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, layout)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
	}
}

func (s *Server) HandleSchemaNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний id")
		return
	}
	var node model.SchemaNode
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний JSON")
		return
	}
	updated, err := s.store.MoveSchemaNode(r.Context(), id, node)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
