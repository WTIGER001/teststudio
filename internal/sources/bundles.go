package sources

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/domain"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Bundle struct {
	ID        string    `json:"id"`
	Project   string    `json:"projectId"`
	Suite     string    `json:"suiteId"`
	Name      string    `json:"name"`
	Manifest  string    `json:"manifestPath"`
	Digest    string    `json:"digest"`
	Content   []byte    `json:"content"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type bundleIndex struct {
	Job       string    `json:"jobId"`
	Suite     string    `json:"suiteId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func bundleBucket(project string) string      { return "source-bundles/" + project }
func buildQueueBucket(location string) string { return "preparation-build-queue/" + location }
func jobQueue(job Job) string {
	if job.Bundle != "" {
		return buildQueueBucket(job.Location)
	}
	return queueBucket(job.Location)
}
func activeJob(state string) bool { return state == "preparing" || state == "discovering" }

func (s *Service) EnqueueBundle(ctx context.Context, user, project, suite, key, name, manifest string, data []byte) (result storage.Record, err error) {
	if err := s.check(ctx, user, project, suite, "manage"); err != nil {
		return result, err
	}
	if !utf8.ValidString(name) || len(name) < 1 || len(name) > 200 || strings.TrimSpace(name) != name || strings.ContainsAny(name, "/\\") || strings.ContainsFunc(name, unicode.IsControl) {
		return result, errors.New("bundle requires a bounded file name")
	}
	if len(key) < 1 || len(key) > 128 || !domain.ValidID(s.Location) {
		return result, errors.New("bundle upload requires an idempotency key")
	}
	snapshot, err := sourcebundle.Validate(data, manifest, suite)
	if err != nil {
		return result, err
	}
	enabled := false
	for _, runner := range s.BundleRunners {
		enabled = enabled || runner == snapshot.Manifest.Runner
	}
	if !enabled {
		return result, errors.New("bundle preparation is not enabled for this runner")
	}
	id := digest("bundle\x00" + user + "\x00" + project + "\x00" + suite + "\x00" + key)[7:]
	err = s.Auth.WithAssignmentTransaction(ctx, func(auth *authorization.Service) error {
		bound := &Service{Store: auth.Directory.Store, Auth: auth}
		if err := bound.check(ctx, user, project, suite, "manage"); err != nil {
			return err
		}
		if err := auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
			return err
		}
		if _, err := (&catalog.Service{Store: bound.Store, Auth: auth}).Tests(ctx, user, project, suite); err != nil {
			return err
		}
		if row, err := bound.Store.Get(ctx, JobBucket(project, suite), id); err == nil {
			job, err := storage.Decode[Job](row)
			if err != nil {
				return err
			}
			if job.SourceDigest != snapshot.Digest || job.Manifest != manifest || job.SourceName != name {
				return errors.New("idempotency key reused for another source bundle")
			}
			result = row
			return nil
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		now := time.Now().UTC()
		expires := now.Add(StagingLifetime)
		job := Job{Kind: "bundle", Bundle: id, SourceDigest: snapshot.Digest, SourceName: name, Manifest: manifest, ID: id, Project: project, Suite: suite, Actor: user, Runner: snapshot.Manifest.Runner, Location: s.Location, State: "received", Phase: "validating", CreatedAt: now}
		bundle := Bundle{ID: id, Project: project, Suite: suite, Name: name, Manifest: manifest, Digest: snapshot.Digest, Content: data, ExpiresAt: expires}
		changes := []storage.Change{}
		for _, item := range []struct {
			bucket, id string
			value      any
		}{{JobBucket(project, suite), id, job}, {bundleBucket(project), id, bundle}, {buildQueueBucket(s.Location), queueID(job), jobIndex{Job: id, Suite: suite, Location: s.Location}}, {"source-bundle-staging", id, bundleIndex{Job: id, Suite: suite, ExpiresAt: expires}}} {
			change, err := storage.Put(item.bucket, item.id, project, 0, item.value)
			if err != nil {
				return err
			}
			changes = append(changes, change)
		}
		audit, err := domain.Audit(user, project, "sources.bundle.upload", suite+"/"+id, nil, map[string]any{"digest": snapshot.Digest, "bytes": len(data)})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, append(changes, audit)...); err != nil {
			return err
		}
		result, err = bound.Store.Get(ctx, JobBucket(project, suite), id)
		return err
	})
	if errors.Is(err, storage.ErrConflict) {
		if row, readErr := s.ReadJob(ctx, user, project, suite, id); readErr == nil {
			job, _ := storage.Decode[Job](row)
			if job.Actor == user && job.SourceDigest == snapshot.Digest && job.Manifest == manifest && job.SourceName == name {
				return row, nil
			}
		}
	}
	return result, err
}

func (s *Service) ReadBundle(ctx context.Context, user, project, suite, id string) (Bundle, error) {
	if err := s.check(ctx, user, project, suite, "read"); err != nil {
		return Bundle{}, err
	}
	row, err := s.Store.Get(ctx, JobBucket(project, suite), id)
	if err != nil {
		return Bundle{}, err
	}
	job, err := storage.Decode[Job](row)
	if err != nil {
		return Bundle{}, err
	}
	if job.Bundle == "" {
		return Bundle{}, storage.ErrNotFound
	}
	return s.loadBundle(ctx, job)
}
func (s *Service) loadBundle(ctx context.Context, job Job) (Bundle, error) {
	row, err := s.Store.Get(ctx, bundleBucket(job.Project), job.Bundle)
	if err != nil {
		return Bundle{}, err
	}
	bundle, err := storage.Decode[Bundle](row)
	if err != nil {
		return bundle, err
	}
	if bundle.ID != job.Bundle || bundle.Project != job.Project || bundle.Suite != job.Suite || bundle.Digest != job.SourceDigest || digest(string(bundle.Content)) != bundle.Digest {
		return Bundle{}, errors.New("stored source bundle does not match its preparation")
	}
	if job.State != "published" && time.Now().After(bundle.ExpiresAt) {
		return Bundle{}, errors.New("source bundle staging has expired")
	}
	return bundle, nil
}
