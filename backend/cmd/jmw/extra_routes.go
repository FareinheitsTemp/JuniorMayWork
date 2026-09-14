package main

import (
	"net/http"
	"strings"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/api"
)

// extraRoutes — реєстрація додаткових маршрутів (grid CRUD, PDF-звіти) поверх
// основних маршрутів сервера, без редагування Routes() у internal/api/misc.go.
func extraRoutes(server *api.Server, allowedOrigin string, wsHandler http.HandlerFunc) http.Handler {
	inner := server.Routes(allowedOrigin, wsHandler)

	extra := http.NewServeMux()
	extra.HandleFunc("GET /api/grid", server.HandleGridTables)
	extra.HandleFunc("GET /api/grid/{table}", server.HandleGridRows)
	extra.HandleFunc("POST /api/grid/{table}", server.HandleGridRowCreate)
	extra.HandleFunc("PATCH /api/grid/{table}/{id}", server.HandleGridRowUpdate)
	extra.HandleFunc("DELETE /api/grid/{table}/{id}", server.HandleGridRowDelete)
	extra.HandleFunc("POST /api/runs/{id}/report", server.HandleRunReportCreate)
	extra.HandleFunc("GET /api/reports/{id}/download", server.HandleReportDownload)
	// Невідомі шляхи (наприклад, /api/reports/summary) ідуть у основний роутер.
	extra.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		inner.ServeHTTP(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/grid") || strings.HasPrefix(p, "/api/runs") || strings.HasPrefix(p, "/api/reports/") {
			extra.ServeHTTP(w, r)
			return
		}
		inner.ServeHTTP(w, r)
	})
}
