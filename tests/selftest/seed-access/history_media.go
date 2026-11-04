package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
	"os"
	"path/filepath"
)

func prepareHistory(ctx context.Context, db *storage.Store, project string) error {
	id := os.Getenv("TESTSTUDIO_SELFTEST_HISTORY_RUN")
	if !domain.ValidID(id) {
		return errors.New("a valid owned history run is required")
	}
	row, err := db.Get(ctx, "runs", id)
	if err != nil {
		return err
	}
	run, err := storage.Decode[runs.Run](row)
	if err != nil {
		return err
	}
	if run.Project != project || run.Request.Suite != "health" || run.State != "completed" || len(run.Tests) != 1 || run.Tests[0].ID != "SELF-CHECK" {
		return errors.New("the history fixture requires its completed native run")
	}
	if os.Getenv("TESTSTUDIO_SELFTEST_HISTORY_REMOVE") == "true" {
		row, err := db.Get(ctx, catalog.Bucket("tests", project, "health"), "SELF-CHECK")
		if err != nil {
			return err
		}
		value, err := storage.Decode[catalog.Test](row)
		if err != nil || value.State != "archived" {
			return errors.New("archive the fixture definition before removal")
		}
		if err := db.Commit(ctx, storage.Change{Record: row, Delete: true}); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]bool{"removed": true})
	}
	files, err := filesystem.New("/tmp/teststudio-artifacts")
	if err != nil {
		return err
	}
	defer files.Close()
	service := &evidence.Service{Store: db, Blobs: files}
	for _, entry := range []struct{ file, name, kind, content string }{{"screenshot.png", "retained-screenshot.png", "screenshot", "image/png"}, {"trace.zip", "retained-trace.zip", "trace", "application/zip"}} {
		data, err := os.ReadFile(filepath.Join("/tmp", "teststudio-diagnostics-"+project, entry.file))
		if err != nil {
			return err
		}
		if len(data) > 8<<20 {
			return errors.New("retained fixture capture is oversized")
		}
		if _, err := service.SaveBytes(ctx, evidence.Artifact{Project: project, Run: id, Test: "SELF-CHECK", Attempt: 1, Kind: entry.kind, Filename: entry.name, ContentType: entry.content}, data, 100<<20); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]bool{"saved": true})
}
