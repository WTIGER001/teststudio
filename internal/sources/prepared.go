package sources

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

func (s *Service) stagePrepared(ctx context.Context, job Job, result preparation.Inspection) (upload Upload, err error) {
	if result.Image != job.Image || result.Project != job.Project || result.Discovery.Manifest.Suite != job.Suite || result.Discovery.Manifest.Runner != job.Runner {
		return upload, errors.New("inspection does not match its project, suite, or image")
	}
	if err := suitespec.ValidateDiscovery(result.Discovery, job.Image); err != nil {
		return upload, err
	}
	if err := s.check(ctx, job.Actor, job.Project, job.Suite, "manage"); err != nil {
		return upload, err
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", job.Project), job.Suite)
	if err != nil {
		return upload, err
	}
	suite, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return upload, err
	}
	if suite.RequiresTarget {
		return upload, errors.New("the local coded profiles require a suite without a target environment")
	}
	if suite.Archived {
		return upload, errors.New("suite is archived")
	}
	for _, application := range result.Discovery.Manifest.Applications {
		if !slices.Contains(suite.Applications, application) {
			return upload, errors.New("manifest application is outside the destination suite")
		}
	}
	content, err := json.Marshal(result.Discovery)
	if err != nil {
		return upload, err
	}
	now := time.Now().UTC()
	id := domain.NewID()
	selectors := map[string]string{}
	for _, test := range result.Discovery.Tests {
		selectors[test.ID] = test.Selector
	}
	artifact := catalog.Artifact{SourceSnapshot: id, SourceProject: job.Project, SourceRevision: digest(string(content)), Image: job.Image, Discovered: selectors, Provenance: &result.Discovery.Origin}
	channel := digest(job.Project + "/" + job.Suite + "/prebuilt")[7:]
	catalogs := &catalog.Service{Store: s.Store, Auth: s.Auth}
	plan, err := catalogs.PreviewPreparedImport(ctx, job.Actor, job.Project, job.Suite, result.Discovery.Catalog, id, channel, artifact)
	if err != nil {
		return upload, err
	}
	upload = Upload{Kind: "prebuilt", Job: job.ID, Artifact: &artifact, ID: id, Project: job.Project, Suite: job.Suite, Name: "prebuilt-" + job.Runner + ".json", Format: "json", Digest: artifact.SourceRevision, Bytes: len(content), Actor: job.Actor, State: "staged", PlanID: plan.ID, Issues: plan.Issues, CreatedAt: now, ExpiresAt: plan.ExpiresAt}
	if len(plan.Issues) > 0 {
		upload.State = "failed"
	}
	row, err := storage.Put(Bucket(job.Project, job.Suite), id, job.Project, 0, upload)
	if err != nil {
		return upload, err
	}
	raw, err := storage.Put("source-content/"+job.Project, id, job.Project, 0, Content{Content: string(content), Digest: upload.Digest})
	if err != nil {
		return upload, err
	}
	staging, err := storage.Put("source-staging", id, job.Project, 0, upload)
	if err != nil {
		return upload, err
	}
	audit, err := domain.Audit(job.Actor, job.Project, "sources.prepare.review", job.Suite+"/"+job.ID, nil, map[string]string{"snapshotId": id, "image": job.Image})
	if err != nil {
		return upload, err
	}
	return upload, s.Store.Commit(ctx, row, raw, staging, audit)
}

func (s *Service) ApproveArtifact(ctx context.Context, artifact catalog.Artifact) error {
	if !domain.ValidID(artifact.SourceProject) || !domain.ValidID(artifact.SourceSnapshot) {
		return authorization.ErrForbidden
	}
	row, err := s.Store.Get(ctx, "prepared-artifacts/"+artifact.SourceProject, artifact.SourceSnapshot)
	if errors.Is(err, storage.ErrNotFound) {
		return authorization.ErrForbidden
	}
	if err != nil {
		return err
	}
	candidate, err := storage.Decode[catalog.Artifact](row)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(candidate, artifact) {
		return nil
	}
	return authorization.ErrForbidden
}
