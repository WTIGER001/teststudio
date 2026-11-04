package sources

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/storage"
)

type EditorFile struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Executable bool   `json:"executable,omitempty"`
}

type PlaywrightDraft struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	ManifestPath string       `json:"manifestPath,omitempty"`
	TargetURL    string       `json:"targetUrl,omitempty"`
	Files        []EditorFile `json:"files"`
}

func draftBucket(project, suite string) string { return "playwright-drafts/" + project + "/" + suite }

func validateDraft(value PlaywrightDraft) error {
	if value.TargetURL != "" && !recording.ValidURL(value.TargetURL) {
		return errors.New("recorded source requires a valid HTTP or HTTPS start address")
	}
	if !domain.ValidID(value.ID) || !utf8.ValidString(value.Name) || strings.TrimSpace(value.Name) != value.Name || len(value.Name) < 1 || len(value.Name) > 120 || strings.ContainsFunc(value.Name, unicode.IsControl) {
		return errors.New("draft requires a permanent identifier and a name of 1 through 120 characters")
	}
	if len(value.Files) < 1 || len(value.Files) > 50 {
		return errors.New("draft requires 1 through 50 text files")
	}
	if value.ManifestPath != "" && !sourcebundle.SafePath(value.ManifestPath) {
		return errors.New("draft manifest must identify a relative source file")
	}
	seen, size := map[string]bool{}, 0
	for _, file := range value.Files {
		key := strings.ToLower(file.Path)
		reserved := key == "teststudio-origin.json" || (value.ManifestPath == "" && (key == "package.json" || key == "package-lock.json" || key == "teststudio-suite.json"))
		if !sourcebundle.SafePath(file.Path) || seen[key] || reserved {
			return errors.New("draft paths must be unique relative files outside generated package, manifest, and provenance files")
		}
		for _, part := range strings.Split(key, "/") {
			if part == "node_modules" {
				return errors.New("drafts use the trusted runtime and must omit node_modules")
			}
		}
		if !utf8.ValidString(file.Content) || strings.ContainsRune(file.Content, 0) || len(file.Content) > 256<<10 {
			return errors.New("draft files require UTF-8 text of at most 256 KiB each")
		}
		seen[key] = true
		size += len(file.Content)
	}
	if size > ContentLimit {
		return errors.New("draft exceeds its 1 MiB text limit")
	}
	for _, file := range value.Files {
		parts := strings.Split(file.Path, "/")
		for i := 1; i < len(parts); i++ {
			if seen[strings.ToLower(strings.Join(parts[:i], "/"))] {
				return errors.New("draft file conflicts with a directory")
			}
		}
	}
	return nil
}

func (s *Service) Drafts(ctx context.Context, user, project, suite, cursor string, limit int) (storage.Page, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return storage.Page{}, err
	}
	if limit < 1 || limit > 100 {
		return storage.Page{}, errors.New("draft page limit must be 1 through 100")
	}
	return s.Store.List(ctx, draftBucket(project, suite), project, cursor, limit)
}

func (s *Service) Draft(ctx context.Context, user, project, suite, id string) (storage.Record, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return storage.Record{}, err
	}
	if !domain.ValidID(id) {
		return storage.Record{}, errors.New("invalid draft identifier")
	}
	return s.Store.Get(ctx, draftBucket(project, suite), id)
}

func (s *Service) SaveDraft(ctx context.Context, user, project, suite string, value PlaywrightDraft, revision int64) (result storage.Record, err error) {
	if err := validateDraft(value); err != nil {
		return result, err
	}
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		for _, permission := range []string{"read", "manage"} {
			if err := bound.check(ctx, user, project, suite, permission); err != nil {
				return err
			}
		}
		if err := bound.draftSuite(ctx, project, suite, false); err != nil {
			return err
		}
		var before any
		if revision > 0 {
			row, err := bound.Store.Get(ctx, draftBucket(project, suite), value.ID)
			if err != nil {
				return err
			}
			before = row.Data
		}
		change, err := storage.Put(draftBucket(project, suite), value.ID, project, revision, value)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.draft.save", suite+"/"+value.ID, nil, map[string]any{"name": value.Name, "files": len(value.Files), "updated": before != nil})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change, audit); err != nil {
			return err
		}
		result, err = bound.Store.Get(ctx, change.Bucket, change.ID)
		return err
	})
	return result, err
}

func (s *Service) draftSuite(ctx context.Context, project, suite string, preparing bool) error {
	row, err := s.Store.Get(ctx, projects.Bucket("suites", project), suite)
	if err != nil {
		return err
	}
	value, err := storage.Decode[projects.Suite](row)
	if err != nil {
		return err
	}
	if value.Archived {
		return errors.New("restore the suite before editing browser source")
	}
	if preparing && value.RequiresTarget {
		return errors.New("the offline browser profile requires a targetless suite")
	}
	return nil
}

func (s *Service) PrepareDraft(ctx context.Context, user, project, suite, id, key string, revision int64) (storage.Record, error) {
	row, err := s.Draft(ctx, user, project, suite, id)
	if err != nil {
		return storage.Record{}, err
	}
	if revision < 1 || row.Revision != revision {
		return storage.Record{}, storage.ErrConflict
	}
	value, err := storage.Decode[PlaywrightDraft](row)
	if err != nil {
		return storage.Record{}, err
	}
	data, err := assembleDraft(suite, value)
	if err != nil {
		return storage.Record{}, err
	}
	return s.enqueueBundle(ctx, user, project, suite, key, "playwright-"+id+".zip", draftManifestPath(value), data, nil, &row)
}
