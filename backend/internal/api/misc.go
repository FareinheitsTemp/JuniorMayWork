package api

import "net/http"

// Routes реєструє основні маршрути REST API v4.
// Grid і додаткові маршрути реєструються в cmd/jmw/extra_routes.go.
func (s *Server) Routes(allowedOrigin string, wsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/orders", s.HandleOrders)
	mux.HandleFunc("GET /api/orders/{id}", s.HandleOrder)
	mux.HandleFunc("PATCH /api/orders/{id}", s.HandleOrder)
	mux.HandleFunc("DELETE /api/orders/{id}", s.HandleOrder)
	mux.HandleFunc("PATCH /api/orders/{id}/status", s.HandleOrderStatus)

	mux.HandleFunc("GET /api/profiles", s.HandleProfiles)
	mux.HandleFunc("POST /api/profiles", s.HandleProfiles)
	mux.HandleFunc("GET /api/profiles/{id}", s.HandleProfile)
	mux.HandleFunc("PATCH /api/profiles/{id}", s.HandleProfile)
	mux.HandleFunc("DELETE /api/profiles/{id}", s.HandleProfile)

	mux.HandleFunc("GET /api/sources", s.HandleSources)
	mux.HandleFunc("GET /api/dashboard", s.HandleDashboard)
	mux.Handle("/ws", wsHandler)

	return cors(allowedOrigin, mux)
}

func cors(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin != "" { w.Header().Set("Access-Control-Allow-Origin", origin) }
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
		next.ServeHTTP(w, r)
	})
}
