package api

import (
	"net/http"
)

// HandleDashboard — GET /api/dashboard: метрики головної сторінки
// (динаміка 30 днів, здоров'я джерел, статуси, підсумки).
func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.GetDashboard(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}
