package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
)

// HandleProfiles — GET /api/profiles, POST /api/profiles.
func (s *Server) HandleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profiles, err := s.store.ListProfiles(r.Context())
		if err != nil { writeStoreErr(w, err); return }
		writeJSON(w, http.StatusOK, profiles)
	case http.MethodPost:
		var input model.Profile
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil { writeErr(w, http.StatusBadRequest, "некоректний JSON"); return }
		profile, err := s.store.CreateProfile(r.Context(), input)
		if err != nil { writeProfileErr(w, err); return }
		writeJSON(w, http.StatusCreated, profile)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleProfile — GET/PATCH/DELETE /api/profiles/{id}.
func (s *Server) HandleProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 { writeErr(w, http.StatusBadRequest, "некоректний id"); return }
	switch r.Method {
	case http.MethodGet:
		profile, err := s.store.GetProfile(r.Context(), id)
		if err != nil { writeProfileErr(w, err); return }
		writeJSON(w, http.StatusOK, profile)
	case http.MethodPatch:
		var input model.Profile
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil { writeErr(w, http.StatusBadRequest, "некоректний JSON"); return }
		profile, err := s.store.UpdateProfile(r.Context(), id, input)
		if err != nil { writeProfileErr(w, err); return }
		writeJSON(w, http.StatusOK, profile)
	case http.MethodDelete:
		if err := s.store.DeleteProfile(r.Context(), id); err != nil { writeProfileErr(w, err); return }
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeProfileErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) { writeErr(w, http.StatusNotFound, "профіль не знайдено"); return }
	writeErr(w, http.StatusBadRequest, err.Error())
}
