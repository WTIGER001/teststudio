package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/passwords"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/sources"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/traceviewer"
)

type Server struct {
	Recordings     *recording.Service
	Sources        *sources.Service
	Store          *storage.Store
	Auth           *authorization.Service
	Projects       *projects.Service
	Catalog        *catalog.Service
	Runs           *runs.Service
	Schedules      *schedules.Service
	Notifications  *notifications.Service
	Evidence       *evidence.Service
	ArtifactOrigin string
	TraceViewer    *traceviewer.Bundle
	Sessions       *sessions.Service
	OIDC           *authoidc.Service
	LocalUser      string
	LocalAccounts  *localauth.Service
	UI             string
}

type sessionKey struct{}

func actor(r *http.Request) string { return r.Context().Value(sessionKey{}).(sessions.Session).UserID }

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func problem(w http.ResponseWriter, err error) {
	status := http.StatusUnprocessableEntity
	if errors.Is(err, passwords.ErrBusy) {
		status = http.StatusTooManyRequests
	}
	if errors.Is(err, sessions.ErrPasswordChangeRequired) {
		status = http.StatusForbidden
	}
	if errors.Is(err, storage.ErrNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, storage.ErrConflict) {
		status = http.StatusConflict
	}
	if errors.Is(err, authorization.ErrForbidden) {
		status = http.StatusForbidden
	}
	if errors.Is(err, sessions.ErrUnauthenticated) {
		status = http.StatusUnauthorized
	}
	if errors.Is(err, evidence.ErrUnavailable) {
		status = http.StatusGone
	}
	JSON(w, status, map[string]any{"status": status, "message": err.Error()})
}

func body(r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return errors.New("request body is not valid for this operation")
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body contains trailing data")
	}
	return nil
}

func revision(r *http.Request) (int64, error) {
	v := r.Header.Get("If-Match")
	if v == "" {
		return 0, errors.New("If-Match revision is required")
	}
	n, err := strconv.ParseInt(strings.Trim(v, "\""), 10, 64)
	if err != nil || n < 0 {
		return 0, errors.New("invalid If-Match revision")
	}
	return n, nil
}

func (s *Server) authorized(handler http.HandlerFunc) http.HandlerFunc {
	return s.authenticated(handler, false)
}

func (s *Server) authenticated(handler http.HandlerFunc, allowPasswordChange bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.Sessions.Read(r.Context(), r)
		if err != nil {
			problem(w, err)
			return
		}
		if v.MustChangePassword && !allowPasswordChange {
			problem(w, sessions.ErrPasswordChangeRequired)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if err := s.Sessions.CheckMutation(r, v); err != nil {
				JSON(w, 403, map[string]string{"message": err.Error()})
				return
			}
		}
		handler(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, v)))
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, 200, map[string]string{"status": "up", "service": "teststudio"})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Store.DB.PingContext(r.Context()); err != nil {
			JSON(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		JSON(w, 200, map[string]string{"status": "up", "service": "teststudio"})
	})
	s.identityLinkRoutes(mux)
	s.authRoutes(mux)
	s.localAccountRoutes(mux)
	s.accessRoutes(mux)
	s.assignmentRoutes(mux)
	s.pendingAccessRoutes(mux)
	s.rolePolicyRoutes(mux)
	s.providerRoutes(mux)
	s.grantRoutes(mux)
	s.builtinRoleRoutes(mux)
	s.permissionExplanationRoutes(mux)
	s.projectRoutes(mux)
	s.catalogRoutes(mux)
	s.testHistoryRoutes(mux)
	s.catalogTransferRoutes(mux)
	s.savedSelectionRoutes(mux)
	s.sourceRoutes(mux)
	s.recordingRoutes(mux)
	s.runRoutes(mux)
	s.scheduleRoutes(mux)
	s.notificationRoutes(mux)
	s.timingRoutes(mux)
	s.trendRoutes(mux)
	s.evidenceRetentionRoutes(mux)
	mux.HandleFunc("GET /api/v1/me/capabilities", s.authorized(func(w http.ResponseWriter, r *http.Request) {
		caps, err := s.Auth.Capabilities(r.Context(), actor(r), r.URL.Query().Get("project"))
		if err != nil {
			problem(w, err)
			return
		}
		JSON(w, 200, caps)
	}))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, 404, map[string]string{"message": "unknown API operation"})
	})
	mux.HandleFunc("/auth/", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, 404, map[string]string{"message": "unknown authentication operation"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.static(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		artifactSource := ""
		frameSource := " 'none'"
		if validOrigin(s.ArtifactOrigin) {
			artifactSource = " " + s.ArtifactOrigin
			frameSource = artifactSource
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:"+artifactSource+"; media-src 'self'"+artifactSource+"; connect-src 'self'"+artifactSource+"; frame-src"+frameSource+"; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/auth/") || strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.UI, filepath.Clean("/"+r.URL.Path))
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	index := filepath.Join(s.UI, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.Error(w, "Build the TestStudio web application to serve the UI.", 503)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, index)
}

func (s *Server) record(w http.ResponseWriter, r *http.Request, bucket, id string) {
	v, err := s.Store.Get(r.Context(), bucket, id)
	if err != nil {
		problem(w, err)
		return
	}
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", v.Revision))
	if bucket == "runs" {
		run, err := storage.Decode[runs.Run](v)
		if err != nil {
			problem(w, err)
			return
		}
		v.Data, err = json.Marshal(runs.NormalizeTimeouts(run))
		if err != nil {
			problem(w, err)
			return
		}
	}
	JSON(w, 200, v)
}
