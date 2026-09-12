// Пакет api: CRUD віток.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

func (s *Server) HandleBranches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListBranches(r.Context(), false)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var b model.Branch
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "некоректний JSON")
			return
		}
		b.Name = strings.TrimSpace(b.Name)
		if b.Name == "" {
			writeErr(w, http.StatusBadRequest, "name обов'язковий")
			return
		}
		if b.Keywords == nil {
			b.Keywords = []string{}
		}
		created, err := s.store.CreateBranch(r.Context(), b)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
	}
}

func (s *Server) HandleBranch(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		b, err := s.store.GetBranch(r.Context(), id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, b)
	case http.MethodPatch:
		var b model.Branch
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "некоректний JSON")
			return
		}
		b.Name = strings.TrimSpace(b.Name)
		if b.Name == "" {
			writeErr(w, http.StatusBadRequest, "name обов'язковий")
			return
		}
		if b.Keywords == nil {
			b.Keywords = []string{}
		}
		updated, err := s.store.UpdateBranch(r.Context(), id, b)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := s.store.DeleteBranch(r.Context(), id); err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "видалено"})
	default:
		w.Header().Set("Allow", "GET, PATCH, DELETE")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
	}
}
