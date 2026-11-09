package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type sourceReadiness struct {
	artifact  Artifact
	revisions map[string]int64
	current   map[string]int64
}

func (s *Service) savedSourceReadiness(ctx context.Context, project string, suite projects.Suite) (sourceReadiness, error) {
	result := sourceReadiness{revisions: map[string]int64{}, current: map[string]int64{}}
	if suite.ActiveRelease == "" {
		return result, nil
	}
	row, err := s.Store.Get(ctx, Bucket("releases", project, suite.ID), suite.ActiveRelease)
	if errors.Is(err, storage.ErrNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	release, err := storage.Decode[Release](row)
	if err != nil {
		return result, err
	}
	result.artifact = release.Draft.Artifact
	if result.artifact.SourceSnapshot == "" {
		return result, nil
	}
	upload, err := s.Store.Get(ctx, "source-uploads/"+project+"/"+suite.ID, result.artifact.SourceSnapshot)
	if errors.Is(err, storage.ErrNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	info, err := storage.Decode[struct {
		Job string `json:"preparationId"`
	}](upload)
	if err != nil || info.Job == "" {
		return result, err
	}
	job, err := s.Store.Get(ctx, "source-jobs/"+project+"/"+suite.ID, info.Job)
	if errors.Is(err, storage.ErrNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	saved, err := storage.Decode[struct {
		Revisions map[string]int64 `json:"savedSourceRevisions"`
	}](job)
	result.revisions = saved.Revisions
	return result, err
}
func (s *Service) sourceReady(ctx context.Context, project, suite string, test Test, ready *sourceReadiness) (bool, error) {
	if test.Runner == "http" {
		return true, nil
	}
	if ready.artifact.Discovered[test.ID] != test.Selector {
		return false, nil
	}
	if !strings.HasPrefix(test.Source, "checks/saved/") {
		return true, nil
	}
	id, _, _ := strings.Cut(strings.TrimPrefix(test.Source, "checks/saved/"), "/")
	revision, ok := ready.current[id]
	if !ok {
		row, err := s.Store.Get(ctx, "playwright-drafts/"+project+"/"+suite, id)
		if errors.Is(err, storage.ErrNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		revision = row.Revision
		ready.current[id] = revision
	}
	return ready.revisions[id] == revision, nil
}
