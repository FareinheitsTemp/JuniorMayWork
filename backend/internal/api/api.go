// Пакет api: REST API бекенду (net/http mux з патернами Go 1.22+).
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/scraper"
	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/store"
)

type Server struct {
	store   *store.Store
	manager *scraper.Manager
	notify  scraper.Notifier
	log     *slog.Logger
}

func NewServer(st *store.Store, m *scraper.Manager, n scraper.Notifier, log *slog.Logger) *Server {
	return &Server{store: st, manager: m, notify: n, log: log}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeStoreErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// parseList: "react, js, next" → ["react","js","next"].
func parseList(s string) []string {
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

var orderStatuses = map[string]bool{
	"new": true, "seen": true, "applied": true,
	"won": true, "lost": true, "archived": true,
}
