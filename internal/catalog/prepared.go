package catalog

import (
	"context"
	"errors"
	"reflect"

	"github.com/wtiger001/teststudio/internal/domain"
)

func MergeManagedMetadata(before, incoming Test) Test {
	incoming.Title, incoming.Description, incoming.Category, incoming.Tags, incoming.Owner = before.Title, before.Description, before.Category, before.Tags, before.Owner
	incoming.State, incoming.Safety, incoming.Rationale, incoming.Conditions = before.State, before.Safety, before.Rationale, before.Conditions
	incoming.TimeoutSeconds, incoming.Timing, incoming.Capture = before.TimeoutSeconds, before.Timing, before.Capture
	return incoming
}

func SameSourceFacts(before, after Test) bool {
	merged := MergeManagedMetadata(before, after)
	return reflect.DeepEqual(before, merged)
}

func (s *Service) PreviewPreparedImport(ctx context.Context, user, project, suite string, document Document, snapshot, source string, artifact Artifact) (ImportPlan, error) {
	if !domain.ValidID(snapshot) || !domain.ValidID(source) || artifact.SourceSnapshot != snapshot || artifact.SourceProject != project {
		return ImportPlan{}, errors.New("prepared catalog requires a project-bound source snapshot")
	}
	for _, permission := range []string{"sources.manage", "catalog.import"} {
		if err := s.Auth.Check(ctx, user, project, "suite/"+suite, permission); err != nil {
			return ImportPlan{}, err
		}
	}
	if document.Schema != SchemaVersion || len(document.Tests) < 1 || len(document.Tests) > 1000 {
		return ImportPlan{}, errors.New("prepared catalogs require the supported schema and 1 through 1000 tests")
	}
	for i := range document.Tests {
		document.Tests[i].DefinitionSource = source
	}
	return s.preview(ctx, user, project, suite, "prepared", document.Tests, nil, snapshot, &artifact, source)
}
