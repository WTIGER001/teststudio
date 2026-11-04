package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

func expireViewer(ctx context.Context, db *storage.Store, project string) error {
	token := os.Getenv("TESTSTUDIO_SELFTEST_VIEWER_TOKEN")
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return errors.New("invalid fixture viewer grant")
	}
	row, err := db.Get(ctx, "artifact-access", sessions.Digest(token))
	if err != nil {
		return err
	}
	value, err := storage.Decode[map[string]any](row)
	if err != nil {
		return err
	}
	if row.Project != project || value["project"] != project || value["traceViewer"] != true {
		return errors.New("fixture viewer belongs to another project or action")
	}
	value["expiresAt"] = time.Now().UTC().Add(-time.Second)
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err != nil {
		return err
	}
	if err := db.Commit(ctx, change); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]bool{"expired": true})
}
