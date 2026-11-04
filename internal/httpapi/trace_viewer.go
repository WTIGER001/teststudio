package httpapi

import (
	"bytes"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Server) tracePolicy() string {
	return "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; font-src 'self' data:; connect-src 'self' blob:; frame-src 'self' blob: data:; worker-src 'self' blob:; manifest-src 'self'; object-src 'none'; base-uri 'self'; form-action 'none'; frame-ancestors 'self' " + s.Sessions.Origin
}

func (s *Server) serveTraceViewer(w http.ResponseWriter, r *http.Request) {
	g, err := s.authorizeArtifactGrant(r.Context(), r.PathValue("token"))
	if err != nil {
		problem(w, err)
		return
	}
	if s.TraceViewer == nil || g.Action != "view" || !g.TraceViewer {
		problem(w, storage.ErrNotFound)
		return
	}
	row, err := s.Store.Get(r.Context(), evidence.Bucket(g.Run), g.Artifact)
	if err != nil {
		problem(w, err)
		return
	}
	a, err := storage.Decode[evidence.Artifact](row)
	if err != nil || a.Kind != "trace" || a.Project != g.Project || a.Run != g.Run {
		problem(w, storage.ErrNotFound)
		return
	}
	available, err := s.Evidence.Available(r.Context(), a, time.Now())
	if err != nil {
		problem(w, err)
		return
	}
	if !available {
		JSON(w, 410, map[string]string{"message": "Trace content is unavailable."})
		return
	}
	if r.PathValue("asset") == "authorization" {
		w.WriteHeader(204)
		return
	}
	if r.PathValue("asset") == "ping" {
		w.WriteHeader(204)
		return
	}
	name := r.PathValue("asset")
	if strings.ContainsAny(name, "\\\x00") || path.Clean(name) != name {
		problem(w, storage.ErrNotFound)
		return
	}
	if name == "index.html" && r.URL.Query().Get("trace") != s.ArtifactOrigin+"/access/"+r.PathValue("token") {
		JSON(w, 422, map[string]string{"message": "The viewer must use its authorized trace."})
		return
	}
	data, ok := s.TraceViewer.Asset(name)
	if !ok {
		problem(w, storage.ErrNotFound)
		return
	}
	w.Header().Set("Content-Security-Policy", s.tracePolicy())
	w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(name)))
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

func (s *Server) traceClosed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self' "+s.Sessions.Origin)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><html lang=\"en\"><title>Trace access ended</title><h1>Trace access ended</h1><p>Return to TestStudio. Close the preview and select View to refresh current access.</p></html>"))
}
