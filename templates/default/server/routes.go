package server

import (
	"fmt"
	"net/http"
)

// RegisterRoutes sets up custom backend API endpoints and middleware.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","framework":"pogonian"}`)
	})
}
