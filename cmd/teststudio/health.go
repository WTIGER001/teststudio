package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func serviceHealthHandler(db *storage.Store, role string) http.Handler {
	mux := http.NewServeMux()
	health := func(ready bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if ready {
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				defer cancel()
				if err := db.DB.PingContext(ctx); err != nil {
					w.WriteHeader(http.StatusServiceUnavailable)
					json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
					return
				}
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "up", "service": "teststudio", "role": role})
		}
	}
	mux.HandleFunc("GET /healthz", health(true))
	mux.HandleFunc("GET /livez", health(false))
	return mux
}
