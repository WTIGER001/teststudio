package httpapi

import (
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
	Session   string    `json:"session"`
	User      string    `json:"user"`
	Project   string    `json:"project"`
	Run       string    `json:"run"`
	Artifact  string    `json:"artifact"`
	Action    string    `json:"action"`
	ExpiresAt time.Time `json:"expiresAt"`
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
	token := sessions.Random()
	grant := artifactGrant{Session: session.ID, User: session.UserID, Project: v.Project, Run: v.ID, Artifact: a.ID, Action: input.Action, ExpiresAt: expires}
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
	JSON(w, 200, map[string]any{"url": s.ArtifactOrigin + "/access/" + token, "expiresAt": expires})
}

func (s *Server) ArtifactHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /access/{token}", s.serveArtifact)
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
	if s.Evidence == nil || len(r.PathValue("token")) != 43 {
		problem(w, storage.ErrNotFound)
		return
	}
	row, err := s.Store.Get(r.Context(), "artifact-access", sessions.Digest(r.PathValue("token")))
	if err != nil {
		problem(w, storage.ErrNotFound)
		return
	}
	g, err := storage.Decode[artifactGrant](row)
	if err != nil || !time.Now().Before(g.ExpiresAt) {
		problem(w, sessions.ErrUnauthenticated)
		return
	}
	session, err := s.Sessions.Validate(r.Context(), g.Session)
	if err != nil || session.MustChangePassword || session.UserID != g.User {
		problem(w, sessions.ErrUnauthenticated)
		return
	}
	permission := "evidence.read"
	if g.Action == "download" {
		permission = "evidence.download"
	}
	if err := s.Auth.Check(r.Context(), g.User, g.Project, "run/"+g.Run, permission); err != nil {
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
