package main

import (
	"context"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/httpapi"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/traceviewer"
)

func serveArtifacts(ctx context.Context, address, origin, artifactOrigin, traceDirectory string, db *storage.Store, auth *authorization.Service, evidenceService *evidence.Service) error {
	var traceBundle *traceviewer.Bundle
	if traceDirectory != "" {
		var err error
		traceBundle, err = traceviewer.Load(traceDirectory)
		if err != nil {
			return err
		}
	}
	sessionService := &sessions.Service{Store: db, Directory: auth.Directory, Origin: origin, LocalAccounts: true}
	api := &httpapi.Server{Store: db, Auth: auth, Sessions: sessionService, Evidence: evidenceService, ArtifactOrigin: artifactOrigin, TraceViewer: traceBundle}
	server := &http.Server{Addr: address, Handler: api.ArtifactHandler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second}
	return serveServers(ctx, []*http.Server{server}, nil)
}
