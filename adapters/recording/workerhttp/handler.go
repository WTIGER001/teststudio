package workerhttp

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/recordingworker"
	"github.com/wtiger001/teststudio/internal/storage"
)

func Handler(worker *recordingworker.Worker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/recording/{id}/{operation}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		contentType, _, contentErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.Header.Values("Origin")) > 0 || len(r.Header.Values("Cookie")) > 0 || len(r.Header.Values("Authorization")) > 0 || len(r.Header.Values("Sec-Fetch-Site")) > 0 || contentErr != nil || contentType != "application/json" {
			w.WriteHeader(403)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "private recording transport required"})
			return
		}
		var request recordingworker.Request
		reader := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		reader.DisallowUnknownFields()
		var extra any
		if reader.Decode(&request) != nil || reader.Decode(&extra) != io.EOF || request.Binding.ID != r.PathValue("id") || (r.PathValue("operation") == "input") != (request.Action != nil) {
			w.WriteHeader(422)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid recording worker request"})
			return
		}
		result, err := worker.Call(r.Context(), r.PathValue("operation"), request)
		if err != nil {
			status := 422
			if errors.Is(err, authorization.ErrForbidden) {
				status = 403
			}
			if errors.Is(err, storage.ErrNotFound) {
				status = 404
			}
			if errors.Is(err, storage.ErrConflict) {
				status = 409
			}
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !worker.Ready(r.Context()) {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	})
	return mux
}
