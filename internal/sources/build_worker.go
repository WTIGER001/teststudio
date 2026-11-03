package sources

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func (s *Service) inspectJob(ctx context.Context, job Job, worker string) (preparation.Inspection, error) {
	var buildLog string
	var buildTruncated bool
	if job.Bundle != "" {
		bundle, err := s.loadBundle(ctx, job)
		if err != nil {
			return preparation.Inspection{}, err
		}
		snapshot, err := sourcebundle.ValidateRoot(bundle.Content, job.Manifest, job.Suite, bundle.Root)
		if err != nil {
			return preparation.Inspection{}, err
		}
		builder := s.Builders[job.Runner]
		origin := suitespec.Origin{SnapshotDigest: snapshot.Digest}
		if bundle.Origin != nil {
			origin = *bundle.Origin
		}
		result, buildErr := builder.Build(ctx, preparation.BuildRequest{Snapshot: snapshot, Origin: origin, Reference: *job.Reference})
		buildLog, buildTruncated = result.Log, result.LogTruncated
		checkpoint, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		job, err = s.checkpointBuild(checkpoint, job, worker, result, buildErr == nil)
		stop()
		if buildErr != nil || err != nil {
			return preparation.Inspection{Log: buildLog, LogTruncated: buildTruncated}, errors.Join(buildErr, err)
		}
	}
	result, err := s.Inspectors[job.Runner].Inspect(ctx, preparation.InspectionRequest{Image: job.Image, Suite: job.Suite, Reference: *job.Reference})
	if len(buildLog) > 128<<10 {
		buildLog = buildLog[:128<<10]
		buildTruncated = true
	}
	if len(result.Log) > 128<<10 {
		result.Log = result.Log[:128<<10]
		result.LogTruncated = true
	}
	result.Log = buildLog + result.Log
	result.LogTruncated = result.LogTruncated || buildTruncated
	return result, err
}

func (s *Service) checkpointBuild(ctx context.Context, claimed Job, worker string, build preparation.Build, succeeded bool) (Job, error) {
	for range 8 {
		row, err := s.Store.Get(ctx, JobBucket(claimed.Project, claimed.Suite), claimed.ID)
		if err != nil {
			return claimed, err
		}
		job, err := storage.Decode[Job](row)
		if err != nil {
			return claimed, err
		}
		if job.Worker != worker || job.State != "preparing" {
			return claimed, storage.ErrConflict
		}
		job.Build = &build
		job.Log, job.LogTruncated = build.Log, build.LogTruncated
		job.Build.Log = ""
		job.Build.LogTruncated = false
		if succeeded {
			if !immutableImage.MatchString(build.Image) || build.SourceDigest != job.SourceDigest {
				return job, errors.New("assembled artifact does not match its stored source digest")
			}
			job.Image = build.Image
			job.State, job.Phase = "discovering", "discovering"
		}
		saved, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, job)
		if err != nil {
			return job, err
		}
		if err := s.Store.Commit(ctx, saved); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return job, err
		}
		if job.CancelRequested {
			return job, context.Canceled
		}
		return job, nil
	}
	return claimed, storage.ErrConflict
}
