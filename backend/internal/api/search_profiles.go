// Пакет api: HTTP CRUD профілів пошуку.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/model"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

func (s *Server) HandleSearchProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profiles, err := s.store.ListSearchProfiles(r.Context())
		if err != nil {
			writeSearchProfileErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, profiles)
	case http.MethodPost:
		var input model.SearchProfile
		if err := decodeSearchProfile(r, &input); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		profile, err := s.store.CreateSearchProfile(r.Context(), input)
		if err != nil {
			writeSearchProfileErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, profile)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
	}
}

func (s *Server) HandleSearchProfile(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "некоректний id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		profile, err := s.store.GetSearchProfile(r.Context(), id)
		if err != nil {
			writeSearchProfileErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, profile)
	case http.MethodPatch:
		var input model.SearchProfile
		if err := decodeSearchProfile(r, &input); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		profile, err := s.store.UpdateSearchProfile(r.Context(), id, input)
		if err != nil {
			writeSearchProfileErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, profile)
	case http.MethodDelete:
		if err := s.store.DeleteSearchProfile(r.Context(), id); err != nil {
			writeSearchProfileErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, PATCH, DELETE")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
	}
}

func decodeSearchProfile(r *http.Request, dst *model.SearchProfile) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("некоректний JSON профілю")
	}
	return nil
}

func writeSearchProfileErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, pgx.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}
