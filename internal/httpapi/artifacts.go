package httpapi

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type artifactGrant struct {
	Session     string    `json:"session"`
	User        string    `json:"user"`
	Project     string    `json:"project"`
	Run         string    `json:"run"`
	Artifact    string    `json:"artifact"`
	Action      string    `json:"action"`
	TraceViewer bool      `json:"traceViewer,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func validOrigin(origin string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}

func (s *Server) artifactAccess(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Action string `json:"action"`
	}
	if err := body(r, &input); err != nil {
		problem(w, err)
		return
	}
	if input.Action == "" {
		input.Action = "view"
	}
	permission := "evidence.read"
	if input.Action == "download" {
		permission = "evidence.download"
	} else if input.Action != "view" {
		problem(w, errors.New("artifact action must be view or download"))
		return
	}
	v, ok := s.authorizedRun(w, r, permission)
	if !ok {
		return
	}
	if s.Evidence == nil || !validOrigin(s.ArtifactOrigin) || s.ArtifactOrigin == s.Sessions.Origin {
		JSON(w, 503, map[string]string{"message": "A separate artifact origin is required."})
		return
	}
	row, err := s.Store.Get(r.Context(), evidence.Bucket(v.ID), r.PathValue("artifact"))
	if err != nil {
		problem(w, err)
		return
	}
	a, err := storage.Decode[evidence.Artifact](row)
	if err != nil {
		problem(w, err)
		return
	}
	if a.Project != v.Project || a.Run != v.ID {
		problem(w, storage.ErrNotFound)
		return
	}
	available, err := s.Evidence.Available(r.Context(), a, time.Now())
	if err != nil {
		problem(w, err)
		return
	}
	if !available {
		JSON(w, 410, map[string]string{"message": "Artifact content is unavailable."})
		return
	}
	session := r.Context().Value(sessionKey{}).(sessions.Session)
	expires := time.Now().UTC().Add(time.Minute)
	if session.ExpiresAt.Before(expires) {
		expires = session.ExpiresAt
	}
	if session.AuthenticationUntil.Before(expires) {
		expires = session.AuthenticationUntil
	}
	viewerAllowed := false
	viewerUnavailable := ""
	if input.Action == "view" && a.Kind == "trace" {
		if s.TraceViewer == nil {
			viewerUnavailable = "The private trace viewer is not installed. Use Download if your current permissions allow it."
		} else if err := s.Evidence.ValidateTrace(r.Context(), v.Project, v.ID, a.ID); err != nil {
			viewerUnavailable = "This file is not a supported Playwright trace or exceeds the viewer limits. Download it for local inspection if permitted."
		} else {
			viewerAllowed = true
		}
	}
	token := sessions.Random()
	grant := artifactGrant{TraceViewer: viewerAllowed, Session: session.ID, User: session.UserID, Project: v.Project, Run: v.ID, Artifact: a.ID, Action: input.Action, ExpiresAt: expires}
	change, err := storage.Put("artifact-access", sessions.Digest(token), v.Project, 0, grant)
	if err != nil {
		problem(w, err)
		return
	}
	audit, err := domain.Audit(session.UserID, v.Project, "evidence.access", v.ID+"/"+a.ID, nil, map[string]any{"action": input.Action, "expiresAt": expires})
	if err == nil {
		err = s.Store.Commit(r.Context(), change, audit)
	}
	if err != nil {
		problem(w, err)
		return
	}
	response := map[string]any{"url": s.ArtifactOrigin + "/access/" + token, "expiresAt": expires}
	if viewerUnavailable != "" {
		response["traceViewerUnavailable"] = viewerUnavailable
	}
	if viewerAllowed {
		query := url.Values{"trace": {s.ArtifactOrigin + "/access/" + token}, "expires": {expires.Format(time.RFC3339)}}
		response["traceViewerUrl"] = s.ArtifactOrigin + "/trace/" + token + "/index.html?" + query.Encode()
	}
	JSON(w, 200, response)
}

func (s *Server) ArtifactHandler() http.Handler {
	mux := http.NewServeMux()
	s.healthRoutes(mux)
	mux.HandleFunc("GET /access/{token}", s.serveArtifact)
	mux.HandleFunc("GET /trace/{token}/{asset...}", s.serveTraceViewer)
	mux.HandleFunc("GET /trace-closed", s.traceClosed)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; frame-ancestors 'none'")
		if r.Header.Get("Origin") == s.Sessions.Origin {
			w.Header().Set("Access-Control-Allow-Origin", s.Sessions.Origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, Content-Length, Content-Range")
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) serveArtifact(w http.ResponseWriter, r *http.Request) {
	g, err := s.authorizeArtifactGrant(r.Context(), r.PathValue("token"))
	if err != nil {
		problem(w, err)
		return
	}
	a, reader, err := s.Evidence.Open(r.Context(), g.Project, g.Run, g.Artifact)
	if err != nil {
		JSON(w, 410, map[string]string{"message": "Artifact content is unavailable."})
		return
	}
	defer reader.Close()
	kind, _, _ := mime.ParseMediaType(a.ContentType)
	inline := g.Action == "view" && (kind == "text/plain" || kind == "application/json" || kind == "image/png" || kind == "image/jpeg" || kind == "image/webp" || kind == "image/gif" || kind == "video/mp4" || kind == "video/webm")
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": a.Filename}))
	contentType := a.ContentType
	if kind == "" || strings.ContainsAny(contentType, "\r\n") {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("ETag", "\""+a.Checksum+"\"")
	if seeker, ok := reader.(io.ReadSeeker); ok {
		http.ServeContent(w, r, a.Filename, a.CreatedAt, seeker)
		return
	}
	if r.Method != "HEAD" {
		_, _ = io.Copy(w, reader)
	}
}

func (s *Server) authorizeArtifactGrant(ctx context.Context, token string) (artifactGrant, error) {
	if s.Evidence == nil || len(token) != 43 {
		return artifactGrant{}, storage.ErrNotFound
	}
	row, err := s.Store.Get(ctx, "artifact-access", sessions.Digest(token))
	if err != nil {
		return artifactGrant{}, storage.ErrNotFound
	}
	g, err := storage.Decode[artifactGrant](row)
	if err != nil || !time.Now().Before(g.ExpiresAt) {
		return g, sessions.ErrUnauthenticated
	}
	session, err := s.Sessions.Validate(ctx, g.Session)
	if err != nil || session.MustChangePassword || session.UserID != g.User {
		return g, sessions.ErrUnauthenticated
	}
	permission := "evidence.read"
	if g.Action == "download" {
		permission = "evidence.download"
	}
	return g, s.Auth.Check(ctx, g.User, g.Project, "run/"+g.Run, permission)
}
