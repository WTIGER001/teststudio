package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) validateSourceSnapshot(ctx context.Context, user, project, suite string, artifact Artifact) error {
	if artifact.SourceSnapshot == "" {
		if artifact.SourceProject != "" {
			return errors.New("project-bound artifacts require a verified source snapshot")
		}
		return nil
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "sources.read"); err != nil {
		return err
	}
	row, err := s.Store.Get(ctx, "source-uploads/"+project+"/"+suite, artifact.SourceSnapshot)
	if err != nil {
		return err
	}
	upload, err := storage.Decode[struct {
		Artifact  *Artifact `json:"artifact,omitempty"`
		Digest    string    `json:"digest"`
		State     string    `json:"state"`
		ExpiresAt time.Time `json:"expiresAt"`
	}](row)
	if err != nil {
		return err
	}
	if artifact.SourceProject != "" && artifact.SourceProject != project {
		return errors.New("artifact belongs to another project")
	}
	if upload.Artifact != nil && !reflect.DeepEqual(*upload.Artifact, artifact) {
		return errors.New("artifact differs from its verified source snapshot")
	}
	if upload.Digest != artifact.SourceRevision || (upload.State != "published" && (upload.State != "staged" || !time.Now().Before(upload.ExpiresAt))) {
		return errors.New("source snapshot is unavailable or does not match the artifact revision")
	}
	row, err = s.Store.Get(ctx, "source-content/"+project, artifact.SourceSnapshot)
	if err != nil {
		return err
	}
	content, err := storage.Decode[struct {
		Content string `json:"content"`
	}](row)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(content.Content))
	if "sha256:"+hex.EncodeToString(hash[:]) != upload.Digest {
		return errors.New("source snapshot digest does not match stored content")
	}
	return nil
}
