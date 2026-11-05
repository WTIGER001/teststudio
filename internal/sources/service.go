package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	AssemblyOnly       bool
	Providers          map[string]sourceprovider.Provider
	Secrets            secrets.Provider
	BundleRunners      []string
	Builders           map[string]preparation.ArtifactBuilder
	RecoveryBuilders   map[string]preparation.ArtifactBuilder
	Inspectors         map[string]preparation.ArtifactInspector
	RecoveryInspectors map[string]preparation.ArtifactInspector
	Location           string
	Store              *storage.Store
	Auth               *authorization.Service
}

func digest(content string) string {
	hash := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func (s *Service) check(ctx context.Context, user, project, suite, permission string) error {
	if !domain.ValidID(project) || !domain.ValidID(suite) {
		return errors.New("invalid project or suite identifier")
	}
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "sources."+permission); err != nil {
		return err
	}
	return s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read")
}

func (s *Service) Stage(ctx context.Context, user, project, suite string, input UploadInput) (result Preview, err error) {
	if err := s.check(ctx, user, project, suite, "manage"); err != nil {
		return result, err
	}
	if !utf8.ValidString(input.Name) || len(input.Name) < 1 || len(input.Name) > 200 || strings.TrimSpace(input.Name) != input.Name || strings.ContainsAny(input.Name, "/\\") || strings.ContainsFunc(input.Name, unicode.IsControl) {
		return result, errors.New("source name must be a file name of up to 200 characters")
	}
	if !utf8.ValidString(input.Content) || len(input.Content) == 0 || len(input.Content) > ContentLimit || (input.Format != "json" && input.Format != "yaml") {
		return result, errors.New("provide a json or yaml source file of at most 1 MiB")
	}
	document, decodeErr := Decode(input.Format, input.Content)
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.check(ctx, user, project, suite, "manage"); err != nil {
			return err
		}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
			return err
		}
		catalogs := &catalog.Service{Store: bound.Store, Auth: auth}
		if _, err := catalogs.Tests(ctx, user, project, suite); err != nil {
			return err
		}
		now := time.Now().UTC()
		upload := Upload{ID: domain.NewID(), Project: project, Suite: suite, Name: input.Name, Format: input.Format, Digest: digest(input.Content), Bytes: len(input.Content), Actor: user, State: "staged", CreatedAt: now, ExpiresAt: now.Add(StagingLifetime), Issues: []catalog.Issue{}}
		if decodeErr != nil {
			upload.State = "failed"
			upload.Issues = append(upload.Issues, catalog.Issue{Field: "content", Message: decodeErr.Error()})
		} else {
			plan, err := catalogs.PreviewSourceImport(ctx, user, project, suite, document, upload.ID)
			if err != nil {
				return err
			}
			upload.PlanID, upload.ExpiresAt, upload.Issues = plan.ID, plan.ExpiresAt, plan.Issues
			if len(plan.Issues) > 0 {
				upload.State = "failed"
			}
			result.Plan = &plan
		}
		row, err := storage.Put(Bucket(project, suite), upload.ID, project, 0, upload)
		if err != nil {
			return err
		}
		content, err := storage.Put("source-content/"+project, upload.ID, project, 0, Content{Content: input.Content, Digest: upload.Digest})
		if err != nil {
			return err
		}
		staged, err := storage.Put("source-staging", upload.ID, project, 0, upload)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(user, project, "sources.upload", suite+"/"+upload.ID, nil, map[string]any{"digest": upload.Digest, "name": upload.Name, "state": upload.State})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, row, content, staged, audit); err != nil {
			return err
		}
		result.Upload, err = bound.Store.Get(ctx, row.Bucket, row.ID)
		return err
	})
	return result, err
}

func (s *Service) Read(ctx context.Context, user, project, suite, id string) (Preview, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return Preview{}, err
	}
	row, err := s.Store.Get(ctx, Bucket(project, suite), id)
	if err != nil {
		return Preview{}, err
	}
	upload, err := storage.Decode[Upload](row)
	if err != nil {
		return Preview{}, err
	}
	if upload.RepositorySource != "" {
		if err := s.checkRepository(ctx, user, project, upload.RepositorySource, "read"); err != nil {
			return Preview{}, err
		}
	}
	result := Preview{Upload: row}
	if upload.PlanID != "" && (upload.State == "published" || time.Now().Before(upload.ExpiresAt)) {
		planRow, err := s.Store.Get(ctx, catalog.Bucket("imports", project, suite), upload.PlanID)
		if err != nil {
			return Preview{}, err
		}
		plan, err := storage.Decode[catalog.ImportPlan](planRow)
		if err != nil {
			return Preview{}, err
		}
		result.Plan = &plan
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, user, project, suite, cursor string, limit int) (storage.Page, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return storage.Page{}, err
	}
	page, err := s.Store.ListNewest(ctx, Bucket(project, suite), project, cursor, limit)
	if err != nil {
		return page, err
	}
	return s.filterRepositoryReads(ctx, user, project, page)
}

func (s *Service) Content(ctx context.Context, user, project, suite, id string) (Content, error) {
	preview, err := s.Read(ctx, user, project, suite, id)
	if err != nil {
		return Content{}, err
	}
	upload, err := storage.Decode[Upload](preview.Upload)
	if err != nil {
		return Content{}, err
	}
	if upload.State != "published" && !time.Now().Before(upload.ExpiresAt) {
		return Content{}, errors.New("staged source has expired")
	}
	row, err := s.Store.Get(ctx, "source-content/"+project, id)
	if err != nil {
		return Content{}, err
	}
	content, err := storage.Decode[Content](row)
	if err == nil && (content.Digest != upload.Digest || digest(content.Content) != upload.Digest) {
		err = errors.New("source snapshot digest does not match stored content")
	}
	return content, err
}
