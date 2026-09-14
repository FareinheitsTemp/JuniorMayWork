package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

// Grid — handlers для гнучкого керування БД (Supabase-style data grid).
// Реєстрація (у Routes, misc.go):
//
//	mux.HandleFunc("GET /api/grid", s.HandleGridTables)
//	mux.HandleFunc("GET /api/grid/{table}", s.HandleGridRows)
//	mux.HandleFunc("POST /api/grid/{table}", s.HandleGridRowCreate)
//	mux.HandleFunc("PATCH /api/grid/{table}/{id}", s.HandleGridRowUpdate)
//	mux.HandleFunc("DELETE /api/grid/{table}/{id}", s.HandleGridRowDelete)

func clampQueryInt(r *http.Request, key string, def, min, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func gridWriteErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "невідома таблиця або рядок")
		return
	}
	writeErr(w, http.StatusBadRequest, err.Error())
}

func (s *Server) HandleGridTables(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tables": store.GridTableNames()})
}

func (s *Server) HandleGridRows(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	limit := clampQueryInt(r, "limit", 50, 1, 100)
	offset := clampQueryInt(r, "offset", 0, 0, 1000000)
	rows, total, columns, err := s.store.GridRows(r.Context(), table,
		limit, offset,
		r.URL.Query().Get("sort"), r.URL.Query().Get("dir"), r.URL.Query().Get("q"))
	if err != nil {
		gridWriteErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rows":     rows,
		"total":    total,
		"columns":  columns,
		"limit":    limit,
		"offset":   offset,
	})
}

func (s *Server) HandleGridRowCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	table := r.PathValue("table")
	var values map[string]any
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		writeErr(w, http.StatusBadRequest, "неправильний JSON")
		return
	}
	if err := s.store.GridInsert(r.Context(), table, values); err != nil {
		gridWriteErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

func (s *Server) HandleGridRowUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	table := r.PathValue("table")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "неправильний id")
		return
	}
	var values map[string]any
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		writeErr(w, http.StatusBadRequest, "неправильний JSON")
		return
	}
	if err := s.store.GridUpdate(r.Context(), table, id, values); err != nil {
		gridWriteErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) HandleGridRowDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", "DELETE")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	table := r.PathValue("table")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "неправильний id")
		return
	}
	if err := s.store.GridDelete(r.Context(), table, id); err != nil {
		gridWriteErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
