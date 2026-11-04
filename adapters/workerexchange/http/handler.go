package httptransport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func Handler(service *workerexchange.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /internal/worker/v1/executions/{execution}/input", func(w http.ResponseWriter, r *http.Request) {
		packet, err := service.Input(r.Context(), r.PathValue("execution"), token(r), r.Header.Get("X-TestStudio-Worker-Instance"))
		if err != nil {
			problem(w, err)
			return
		}
		json.NewEncoder(w).Encode(packet)
	})
	mux.HandleFunc("POST /internal/worker/v1/executions/{execution}/started", func(w http.ResponseWriter, r *http.Request) {
		var event struct {
			At time.Time `json:"at"`
		}
		if err := read(w, r, &event, 1024); err != nil {
			problem(w, err)
			return
		}
		if err := service.Started(r.Context(), r.PathValue("execution"), token(r), r.Header.Get("X-TestStudio-Worker-Instance"), event.At); err != nil {
			problem(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PUT /internal/worker/v1/executions/{execution}/report", func(w http.ResponseWriter, r *http.Request) {
		var submission workerexchange.Submission
		if err := read(w, r, &submission, workerexchange.MaxSubmissionBytes); err != nil {
			problem(w, err)
			return
		}
		if err := service.Submit(r.Context(), r.PathValue("execution"), token(r), r.Header.Get("X-TestStudio-Worker-Instance"), submission); err != nil {
			problem(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PUT /internal/worker/v1/executions/{execution}/attachments/{file}", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
		if err := service.Upload(r.Context(), r.PathValue("execution"), token(r), r.Header.Get("X-TestStudio-Worker-Instance"), r.PathValue("file"), r.Body); err != nil {
			problem(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.URL.RawQuery != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Origin") != "" {
			problem(w, workerexchange.ErrDenied)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func token(r *http.Request) string {
	if len(r.Header.Values("Authorization")) != 1 {
		return ""
	}
	value, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !found {
		return ""
	}
	return value
}

func read(w http.ResponseWriter, r *http.Request, value any, limit int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(new(any)) != io.EOF {
		return workerexchange.ErrInvalid
	}
	return nil
}

func problem(w http.ResponseWriter, err error) {
	status, message := http.StatusServiceUnavailable, workerexchange.ErrUnavailable.Error()
	switch {
	case errors.Is(err, workerexchange.ErrDenied):
		status, message = http.StatusNotFound, workerexchange.ErrDenied.Error()
	case errors.Is(err, workerexchange.ErrInvalid):
		status, message = http.StatusBadRequest, workerexchange.ErrInvalid.Error()
	case errors.Is(err, workerexchange.ErrQuota):
		status, message = http.StatusInsufficientStorage, workerexchange.ErrQuota.Error()
	case errors.Is(err, workerexchange.ErrConflict):
		status, message = http.StatusConflict, workerexchange.ErrConflict.Error()
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
