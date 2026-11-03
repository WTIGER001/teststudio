package catalog

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Export(ctx context.Context, user, project, suite string) (Document, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.export"); err != nil {
		return Document{}, err
	}
	rows, err := s.Tests(ctx, user, project, suite)
	if err != nil {
		return Document{}, err
	}
	now := time.Now().UTC()
	d := Document{Schema: SchemaVersion, Project: project, Suite: suite, ExportedAt: &now, Tests: []Test{}}
	for _, row := range rows {
		t, err := storage.Decode[Test](row)
		if err != nil {
			return d, err
		}
		d.Tests = append(d.Tests, t)
	}
	return d, nil
}

func (s *Service) PreviewImport(ctx context.Context, user, project, suite string, document Document) (ImportPlan, error) {
	return s.previewImport(ctx, user, project, suite, document, "")
}

func (s *Service) PreviewSourceImport(ctx context.Context, user, project, suite string, document Document, snapshot string) (ImportPlan, error) {
	if !domain.ValidID(snapshot) {
		return ImportPlan{}, errors.New("valid source snapshot identifier is required")
	}
	return s.previewImport(ctx, user, project, suite, document, snapshot)
}

func (s *Service) previewImport(ctx context.Context, user, project, suite string, document Document, snapshot string) (ImportPlan, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
		return ImportPlan{}, err
	}
	if document.Schema != SchemaVersion {
		return ImportPlan{}, errors.New("unsupported catalog schema version")
	}
	if len(document.Tests) < 1 || len(document.Tests) > MutationLimit {
		return ImportPlan{}, fmt.Errorf("an import accepts between 1 and %d test definitions", MutationLimit)
	}
	return s.preview(ctx, user, project, suite, "import", document.Tests, nil, snapshot, nil, "")
}

func (s *Service) PreviewBulk(ctx context.Context, user, project, suite string, input BulkRequest) (ImportPlan, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.read"); err != nil {
		return ImportPlan{}, err
	}
	if !slices.Contains([]string{"disable", "enable", "archive", "restore"}, input.Action) {
		return ImportPlan{}, errors.New("unsupported bulk lifecycle action")
	}
	if len(input.Tests) < 1 || len(input.Tests) > MutationLimit {
		return ImportPlan{}, fmt.Errorf("a bulk action accepts between 1 and %d tests", MutationLimit)
	}
	tests := make([]Test, 0, len(input.Tests))
	revisions := map[string]int64{}
	for _, item := range input.Tests {
		if item.Revision < 1 || revisions[item.ID] > 0 {
			return ImportPlan{}, errors.New("bulk actions require unique identifiers and positive revisions")
		}
		r, err := s.Store.Get(ctx, Bucket("tests", project, suite), item.ID)
		if err != nil {
			return ImportPlan{}, err
		}
		if r.Revision != item.Revision {
			return ImportPlan{}, storage.ErrConflict
		}
		t, err := storage.Decode[Test](r)
		if err != nil {
			return ImportPlan{}, err
		}
		t.State = map[string]string{"disable": "disabled", "enable": "enabled", "archive": "archived", "restore": "enabled"}[input.Action]
		tests = append(tests, t)
		revisions[item.ID] = item.Revision
	}
	return s.preview(ctx, user, project, suite, input.Action, tests, revisions, "", nil, "")
}

func (s *Service) preview(ctx context.Context, user, project, suite, kind string, tests []Test, revisions map[string]int64, snapshot string, artifact *Artifact, source string) (ImportPlan, error) {
	v, suiteRow, err := s.suite(ctx, project, suite)
	if err != nil {
		return ImportPlan{}, err
	}
	head, err := s.head(ctx, project, suite)
	if err != nil {
		return ImportPlan{}, err
	}
	rows, err := s.Tests(ctx, user, project, suite)
	if err != nil {
		return ImportPlan{}, err
	}
	current := map[string]storage.Record{}
	merged := map[string]Test{}
	for _, row := range rows {
		t, err := storage.Decode[Test](row)
		if err != nil {
			return ImportPlan{}, err
		}
		current[t.ID] = row
		merged[t.ID] = t
	}
	plan := ImportPlan{Artifact: artifact, SourceSnapshot: snapshot, ID: domain.NewID(), Project: project, Suite: suite, Actor: user, Kind: kind, HeadRevision: head.Revision, SuiteRevision: suiteRow.Revision, Changes: []PlannedTest{}, Issues: []Issue{}, CreatedAt: time.Now().UTC()}
	plan.ExpiresAt = plan.CreatedAt.Add(30 * time.Minute)
	seen := map[string]bool{}
	for _, test := range tests {
		if seen[test.ID] {
			plan.Issues = append(plan.Issues, Issue{TestID: test.ID, Field: "id", Message: "duplicate identifier in request"})
			continue
		}
		seen[test.ID] = true
		if kind == "import" && test.Runner != "http" {
			plan.Issues = append(plan.Issues, Issue{TestID: test.ID, Field: "runner", Message: "declarative import accepts native HTTP definitions; use source preparation for coded tests"})
		}
		change := PlannedTest{Test: test, Revision: current[test.ID].Revision, Action: "create"}
		if row, ok := current[test.ID]; ok {
			before, err := storage.Decode[Test](row)
			if err != nil {
				return plan, err
			}
			if kind == "prepared" {
				if before.DefinitionSource != "" && before.DefinitionSource != source {
					plan.Issues = append(plan.Issues, Issue{TestID: test.ID, Field: "source", Message: "test belongs to another registered source"})
				}
				test = MergeManagedMetadata(before, test)
				change.Test = test
			} else if before.DefinitionSource != "" && !SameSourceFacts(before, test) {
				plan.Issues = append(plan.Issues, Issue{TestID: test.ID, Field: "source", Message: "change source-owned definitions through their registered source"})
			}
			change.Before = &before
			change.Action = "update"
			if revisions != nil && revisions[test.ID] != row.Revision {
				return plan, storage.ErrConflict
			}
			if kind == "import" && test.State != before.State {
				plan.Issues = append(plan.Issues, Issue{TestID: test.ID, Field: "state", Message: "use an explicit lifecycle action to change an existing test state"})
			}
			if reflect.DeepEqual(before, test) {
				change.Action = "unchanged"
			}
		}
		if test.Application != "" && !slices.Contains(v.Applications, test.Application) {
			plan.Issues = append(plan.Issues, Issue{TestID: test.ID, Field: "applicationId", Message: "test application is outside the suite"})
		}
		if err := s.authorizeChange(ctx, user, project, suite, kind, change); err != nil {
			return plan, err
		}
		plan.Changes = append(plan.Changes, change)
		merged[test.ID] = test
	}
	if kind == "prepared" {
		for id, row := range current {
			before := merged[id]
			if seen[id] || before.DefinitionSource != source || before.State == "archived" {
				continue
			}
			after := before
			after.State = "archived"
			change := PlannedTest{Test: after, Before: &before, Revision: row.Revision, Action: "archive"}
			if err := s.authorizeChange(ctx, user, project, suite, kind, change); err != nil {
				return plan, err
			}
			plan.Changes = append(plan.Changes, change)
			merged[id] = after
		}
	}
	all := make([]Test, 0, len(merged))
	for _, test := range merged {
		if kind == "prepared" && test.State == "archived" {
			continue
		}
		all = append(all, test)
	}
	slices.SortFunc(all, func(a, b Test) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	validation := []Issue{}
	if kind == "import" || kind == "prepared" {
		draft := Draft{Tests: all, Capture: DefaultCapture()}
		if artifact != nil {
			draft.Artifact = *artifact
		}
		validation = Validate(draft)
	}
	for _, issue := range validation {
		if kind != "import" && kind != "prepared" && (issue.Field == "artifact.image" || issue.Field == "selector") {
			continue
		}
		if kind == "import" && !seen[issue.TestID] && (issue.Field == "artifact.image" || issue.Field == "selector") {
			continue
		}
		plan.Issues = append(plan.Issues, issue)
	}
	after, err := s.head(ctx, project, suite)
	if err != nil {
		return plan, err
	}
	if after.Revision != head.Revision {
		return plan, storage.ErrConflict
	}
	entry, err := storage.Put(Bucket("imports", project, suite), plan.ID, project, 0, plan)
	if err != nil {
		return plan, err
	}
	return plan, s.Store.Commit(ctx, entry)
}

func (s *Service) authorizeChange(ctx context.Context, user, project, suite, kind string, c PlannedTest) error {
	permission := "catalog.update"
	if c.Revision == 0 {
		permission = "catalog.create"
	}
	if kind == "archive" || c.Action == "archive" {
		permission = "catalog.archive"
	}
	if kind == "restore" || kind == "enable" {
		permission = "catalog.restore"
	}
	if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+c.Test.ID, permission); err != nil {
		return err
	}
	if c.Before != nil && c.Before.Safety != c.Test.Safety {
		return s.Auth.Check(ctx, user, project, "test/"+suite+"/"+c.Test.ID, "catalog.publish")
	}
	return nil
}

func (s *Service) ApplyImport(ctx context.Context, user, project, suite, id string, revision int64) (ImportPlan, error) {
	return s.applyImport(ctx, user, project, suite, id, revision, "")
}

func (s *Service) ApplySourceImport(ctx context.Context, user, project, suite, id string, revision int64, snapshot string) (ImportPlan, error) {
	if !domain.ValidID(snapshot) {
		return ImportPlan{}, errors.New("valid source snapshot identifier is required")
	}
	return s.applyImport(ctx, user, project, suite, id, revision, snapshot)
}

func (s *Service) applyImport(ctx context.Context, user, project, suite, id string, revision int64, snapshot string) (result ImportPlan, resultErr error) {
	defer func() {
		if !errors.Is(resultErr, storage.ErrConflict) {
			return
		}
		current, err := s.Store.Get(ctx, Bucket("imports", project, suite), id)
		if err != nil {
			return
		}
		applied, err := storage.Decode[ImportPlan](current)
		if err == nil && applied.SourceSnapshot == snapshot && applied.Actor == user && applied.Project == project && applied.Suite == suite && applied.AppliedAt != nil {
			result = applied
			resultErr = nil
		}
	}()
	r, err := s.Store.Get(ctx, Bucket("imports", project, suite), id)
	if err != nil {
		return ImportPlan{}, err
	}
	plan, err := storage.Decode[ImportPlan](r)
	if err != nil {
		return plan, err
	}
	if plan.SourceSnapshot != snapshot || plan.Actor != user || plan.Project != project || plan.Suite != suite {
		return plan, authorization.ErrForbidden
	}
	if plan.Kind == "import" || plan.Kind == "prepared" {
		if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.import"); err != nil {
			return plan, err
		}
	}
	for _, c := range plan.Changes {
		if err := s.authorizeChange(ctx, user, project, suite, plan.Kind, c); err != nil {
			return plan, err
		}
	}
	if plan.AppliedAt != nil {
		return plan, nil
	}
	if r.Revision != revision {
		return plan, storage.ErrConflict
	}
	if !time.Now().Before(plan.ExpiresAt) || len(plan.Issues) > 0 {
		return plan, errors.New("catalog plan expired or contains validation errors")
	}
	head, err := s.head(ctx, project, suite)
	if err != nil {
		return plan, err
	}
	if head.Revision != plan.HeadRevision {
		return plan, storage.ErrConflict
	}
	_, suiteRow, err := s.suite(ctx, project, suite)
	if err != nil {
		return plan, err
	}
	if suiteRow.Revision != plan.SuiteRevision {
		return plan, storage.ErrConflict
	}
	changes := []storage.Change{}
	for _, c := range plan.Changes {
		if c.Action == "unchanged" {
			continue
		}
		entry, err := storage.Put(Bucket("tests", project, suite), c.Test.ID, project, c.Revision, c.Test)
		if err != nil {
			return plan, err
		}
		history, err := storage.Put(Bucket("test-revisions", project, suite)+"/"+c.Test.ID, fmt.Sprintf("%020d", c.Revision+1), project, 0, Revision{Test: c.Test, Actor: user, At: time.Now().UTC()})
		if err != nil {
			return plan, err
		}
		audit, err := domain.Audit(user, project, "catalog."+plan.Kind, suite+"/"+c.Test.ID, c.Before, c.Test)
		if err != nil {
			return plan, err
		}
		changes = append(changes, entry, history, audit)
	}
	guard, err := headChange(project, suite, head.Revision)
	if err != nil {
		return plan, err
	}
	now := time.Now().UTC()
	plan.AppliedAt = &now
	applied, err := storage.Put(r.Bucket, r.ID, project, r.Revision, plan)
	if err != nil {
		return plan, err
	}
	suiteGuard := storage.Change{Record: suiteRow}
	changes = append(changes, guard, applied, suiteGuard)
	err = s.Store.CommitAll(ctx, changes...)
	if errors.Is(err, storage.ErrConflict) {
		if current, readErr := s.Store.Get(ctx, r.Bucket, r.ID); readErr == nil {
			if applied, decodeErr := storage.Decode[ImportPlan](current); decodeErr == nil && applied.AppliedAt != nil {
				return applied, nil
			}
		}
	}
	return plan, err
}

func (s *Service) Duplicate(ctx context.Context, user, project, suite, id, newID string, revision int64) error {
	if err := s.Auth.Check(ctx, user, project, "test/"+suite+"/"+id, "catalog.read"); err != nil {
		return err
	}
	r, err := s.Store.Get(ctx, Bucket("tests", project, suite), id)
	if err != nil {
		return err
	}
	if r.Revision != revision {
		return storage.ErrConflict
	}
	t, err := storage.Decode[Test](r)
	if err != nil {
		return err
	}
	t.ID = newID
	t.DefinitionSource = ""
	t.Title += " (copy)"
	t.State = "draft"
	return s.SaveTest(ctx, user, project, suite, t, 0)
}
