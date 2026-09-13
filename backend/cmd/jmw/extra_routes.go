package main

import (
	"net/http"
	"strings"

	"github.com/FareinheitsTemp/JuniorMayWork/backend/internal/api"
)

// extraRoutes — реєстрація додаткових маршрутів (grid CRUD) поверх основних
// маршрутів сервера, без редагування Routes() у internal/api/misc.go.
func extraRoutes(server *api.Server, allowedOrigin string, wsHandler http.HandlerFunc) http.Handler {
	inner := server.Routes(allowedOrigin, wsHandler)

	extra := http.NewServeMux()
	extra.HandleFunc("GET /api/grid", server.HandleGridTables)
	extra.HandleFunc("GET /api/grid/{table}", server.HandleGridRows)
	extra.HandleFunc("POST /api/grid/{table}", server.HandleGridRowCreate)
	extra.HandleFunc("PATCH /api/grid/{table}/{id}", server.HandleGridRowUpdate)
	extra.HandleFunc("DELETE /api/grid/{table}/{id}", server.HandleGridRowDelete)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/grid") {
			extra.ServeHTTP(w, r)
			return
		}
		inner.ServeHTTP(w, r)
	})
}
