package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/wtiger001/teststudio/internal/reliability"
	"github.com/wtiger001/teststudio/internal/storage"
)

func checkParameters(r *http.Request) (reliability.Query, string, int, error) {
	params := r.URL.Query()
	q := reliability.Query{Scope: reliability.Scope{Project: r.PathValue("project"), Suite: r.PathValue("suite"), Test: r.PathValue("test"), Environment: params.Get("environment"), Location: params.Get("location")}}
	var err error
	q.Since, err = time.Parse(time.RFC3339Nano, params.Get("since"))
	if err != nil {
		return q, "", 0, errors.New("individual checks require since and until timestamps")
	}
	q.Until, err = time.Parse(time.RFC3339Nano, params.Get("until"))
	if err != nil || !q.Since.Before(q.Until) || q.Until.After(time.Now().Add(time.Second)) {
		return q, "", 0, errors.New("individual checks require ordered past boundaries")
	}
	limit := 50
	if params.Has("limit") {
		limit, err = strconv.Atoi(params.Get("limit"))
	}
	if err != nil || limit < 1 || limit > 100 {
		return q, "", 0, errors.New("individual checks limit must be from 1 through 100")
	}
	return q, params.Get("cursor"), limit, nil
}

func (s *Server) reliabilityCheckRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/v1/projects/{project}/suites/{suite}/tests/{test}/reliability/checks", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		q, cursor, limit, err := checkParameters(r)
		if err != nil {
			problem(w, err)
			return
		}
		page, err := s.Runs.ReliabilityChecks(r.Context(), actor(r), q, cursor, limit)
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, page)
	}))
	slots := make(chan struct{}, 8)
	m.HandleFunc("GET /api/v1/public/status-pages/{token}/checks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		q, cursor, limit, err := checkParameters(r)
		if err != nil {
			problem(w, err)
			return
		}
		row, err := strconv.Atoi(r.URL.Query().Get("row"))
		if err != nil || row < 0 {
			JSON(w, 422, map[string]string{"message": "select a published row"})
			return
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			w.Header().Set("Retry-After", "5")
			JSON(w, 503, map[string]string{"message": "Status information is busy. Retry shortly."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		page, err := s.statusPageService().PublicChecks(ctx, r.PathValue("token"), row, q, cursor, limit)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				JSON(w, 404, map[string]string{"message": "This status page is unavailable."})
				return
			}
			JSON(w, 422, map[string]string{"message": "Individual checks are unavailable. Refresh the status page and try again."})
			return
		}
		JSON(w, 200, page)
	})
	m.HandleFunc("OPTIONS /api/v1/public/status-pages/{token}/checks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.WriteHeader(http.StatusNoContent)
	})
}
