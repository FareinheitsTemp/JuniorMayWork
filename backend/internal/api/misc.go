// Пакет api: події, заявки, статистика, звіти, керування скрейпером, маршрути.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) HandleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	list, err := s.store.ListEvents(r.Context(), limit)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) HandleApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	list, err := s.store.ListApplications(r.Context(), limit)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) HandleApplication(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний id")
		return
	}
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	var body struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "некоректний JSON")
		return
	}
	switch body.Result {
	case "pending", "accepted", "declined":
	default:
		writeErr(w, http.StatusBadRequest, "result має бути pending/accepted/declined")
		return
	}
	app, err := s.store.SetApplicationResult(r.Context(), id, body.Result)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.Stats(r.Context())
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) HandleReport(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := now.AddDate(0, 0, -30)
	to := now
	q := r.URL.Query()
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t.Add(24 * time.Hour) // включно
		}
	}
	rep, err := s.store.ReportSummary(r.Context(), from, to)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) HandleScraperRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	s.manager.RunOnce(r.Context())
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "збір запущено"})
}

func (s *Server) HandleScraperStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeErr(w, http.StatusMethodNotAllowed, "метод не підтримується")
		return
	}
	writeJSON(w, http.StatusOK, s.manager.Status())
}

// Routes збирає всі маршрути + CORS для фронтенду + точку /ws.
func (s *Server) Routes(allowedOrigin string, wsHandler http.HandlerFunc) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/branches", s.HandleBranches)
	mux.HandleFunc("POST /api/branches", s.HandleBranches)
	mux.HandleFunc("GET /api/branches/{id}", s.HandleBranch)
	mux.HandleFunc("PATCH /api/branches/{id}", s.HandleBranch)
	mux.HandleFunc("DELETE /api/branches/{id}", s.HandleBranch)
	mux.HandleFunc("GET /api/orders", s.HandleOrders)
	mux.HandleFunc("GET /api/orders/{id}", s.HandleOrder)
	mux.HandleFunc("PATCH /api/orders/{id}", s.HandleOrder)
	mux.HandleFunc("DELETE /api/orders/{id}", s.HandleOrder)
	mux.HandleFunc("POST /api/orders/{id}/apply", s.HandleApply)
	mux.HandleFunc("GET /api/events", s.HandleEvents)
	mux.HandleFunc("GET /api/applications", s.HandleApplications)
	mux.HandleFunc("PATCH /api/applications/{id}", s.HandleApplication)
	mux.HandleFunc("GET /api/stats", s.HandleStats)
	mux.HandleFunc("GET /api/reports/summary", s.HandleReport)
	mux.HandleFunc("POST /api/scraper/run", s.HandleScraperRun)
	mux.HandleFunc("GET /api/scraper/status", s.HandleScraperStatus)
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if wsHandler != nil {
		mux.HandleFunc("GET /ws", wsHandler)
	}
	return cors(allowedOrigin, mux)
}

func cors(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
