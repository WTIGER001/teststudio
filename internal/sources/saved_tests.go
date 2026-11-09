package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type savedTestRequest struct {
	Actor    string `json:"actor"`
	Suite    string `json:"suiteId"`
	Draft    string `json:"sourceId"`
	Revision int64  `json:"revision"`
	Error    string `json:"error,omitempty"`
}

func savedTestQueue(location string) string { return "saved-test-queue/" + location }
func savedTestPrefix(id string) string      { return "checks/saved/" + id + "/" }
func savedTestDocument(value PlaywrightDraft) (catalog.Document, string, error) {
	path := "catalog.json"
	if value.ManifestPath != "" {
		for _, f := range value.Files {
			if f.Path == value.ManifestPath {
				var m struct {
					Catalog string `json:"catalog"`
				}
				if err := json.Unmarshal([]byte(f.Content), &m); err != nil {
					return catalog.Document{}, "", err
				}
				path = m.Catalog
			}
		}
	}
	for _, f := range value.Files {
		if f.Path == path {
			var d catalog.Document
			err := json.Unmarshal([]byte(f.Content), &d)
			if err != nil {
				return d, path, err
			}
			if d.Schema != catalog.SchemaVersion || len(d.Tests) < 1 {
				return d, path, errors.New("include at least one test before saving")
			}
			return d, path, nil
		}
	}
	return catalog.Document{}, path, errors.New("the saved test requires its catalog definition")
}
func (s *Service) registerSavedTests(ctx context.Context, user, project, suite string, value PlaywrightDraft, revision int64) ([]storage.Change, error) {
	if err := s.Auth.Check(ctx, user, project, "suite/"+suite, "catalog.publish"); err != nil {
		return nil, err
	}
	d, _, err := savedTestDocument(value)
	if err != nil {
		return nil, err
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", project), suite)
	if err != nil {
		return nil, err
	}
	definition, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return nil, err
	}
	c := &catalog.Service{Store: s.Store, Auth: s.Auth}
	seen := map[string]bool{}
	for _, test := range d.Tests {
		if seen[test.ID] {
			return nil, errors.New("test identifiers must be unique")
		}
		seen[test.ID] = true
		if test.Runner != "playwright" {
			return nil, errors.New("this source editor saves Playwright tests")
		}
		if test.State == "draft" {
			test.State = "enabled"
		}
		if test.Application == "" {
			test.Application = value.Application
			if test.Application == "" && len(definition.Applications) == 1 {
				test.Application = definition.Applications[0]
			}
		}
		test.Source = savedTestPrefix(value.ID) + test.Source
		old, err := s.Store.Get(ctx, catalog.Bucket("tests", project, suite), test.ID)
		if err == nil {
			current, e := storage.Decode[catalog.Test](old)
			if e != nil {
				return nil, e
			}
			if !strings.HasPrefix(current.Source, savedTestPrefix(value.ID)) {
				return nil, errors.New("that test identifier is already used by another test")
			}
			if e = s.Auth.Check(ctx, user, project, "test/"+suite+"/"+test.ID, "catalog.update"); e != nil {
				return nil, e
			}
		} else if errors.Is(err, storage.ErrNotFound) {
			if err = c.SaveTest(ctx, user, project, suite, test, 0); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	request := savedTestRequest{Actor: user, Suite: suite, Draft: value.ID, Revision: revision}
	id := digest(project + "/" + suite + "/" + value.ID)[7:]
	row, err := s.Store.Get(ctx, savedTestQueue(s.Location), id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	change, err := storage.Put(savedTestQueue(s.Location), id, project, row.Revision, request)
	return []storage.Change{change}, err
}
func (s *Service) savedSourceRows(ctx context.Context, project, suite string) ([]storage.Record, error) {
	rows := []storage.Record{}
	cursor := ""
	size, scanned := 0, 0
	for {
		page, err := s.Store.List(ctx, draftBucket(project, suite), project, cursor, 25)
		if err != nil {
			return nil, err
		}
		for _, row := range page.Items {
			scanned++
			if scanned > 4096 {
				return nil, errors.New("this suite exceeds the source record limit")
			}
			value, err := storage.Decode[PlaywrightDraft](row)
			if err != nil {
				return nil, err
			}
			if !value.SaveAsTest {
				continue
			}
			if err = validateDraft(value); err != nil {
				return nil, err
			}
			size += len(row.Data)
			if size > 16<<20 || len(rows) >= 1000 {
				return nil, errors.New("saved source exceeds the suite setup limit")
			}
			rows = append(rows, row)
		}
		if page.Cursor == "" {
			return rows, nil
		}
		cursor = page.Cursor
	}
}
func (s *Service) combinedSavedTests(ctx context.Context, user, project, suite string, primary storage.Record) (PlaywrightDraft, map[string]int64, error) {
	value, err := storage.Decode[PlaywrightDraft](primary)
	if err != nil {
		return value, nil, err
	}
	rows, err := s.savedSourceRows(ctx, project, suite)
	if err != nil {
		return value, nil, err
	}
	combined := value
	combined.Files = nil
	document := catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{}}
	revisions := map[string]int64{}
	_, catalogPath, err := savedTestDocument(value)
	if err != nil {
		return value, nil, err
	}
	for _, f := range value.Files {
		if f.Path == value.ManifestPath || f.Path == "playwright.config.ts" {
			combined.Files = append(combined.Files, f)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	for _, row := range rows {
		saved, e := storage.Decode[PlaywrightDraft](row)
		if e != nil {
			return value, nil, e
		}
		if !saved.SaveAsTest {
			continue
		}
		if saved.ManifestPath != value.ManifestPath || (saved.TargetURL == "") != (value.TargetURL == "") {
			return value, nil, errors.New("saved tests in one suite must use the same execution profile")
		}
		d, path, e := savedTestDocument(saved)
		if e != nil {
			return value, nil, e
		}
		for _, f := range saved.Files {
			if f.Path == saved.ManifestPath || f.Path == "playwright.config.ts" {
				for _, base := range combined.Files {
					if base.Path == f.Path && base.Content != f.Content {
						return value, nil, errors.New("saved tests in one suite require compatible browser settings")
					}
				}
				continue
			}
			if f.Path == path {
				continue
			}
			f.Path = savedTestPrefix(saved.ID) + f.Path
			combined.Files = append(combined.Files, f)
		}
		for _, test := range d.Tests {
			current, e := s.Store.Get(ctx, catalog.Bucket("tests", project, suite), test.ID)
			if e != nil {
				return value, nil, e
			}
			metadata, e := storage.Decode[catalog.Test](current)
			if e != nil {
				return value, nil, e
			}
			test = catalog.MergeManagedMetadata(metadata, test)
			test.Application = metadata.Application
			test.Source = savedTestPrefix(saved.ID) + test.Source
			test.DefinitionSource = ""
			document.Tests = append(document.Tests, test)
		}
		revisions[row.ID] = row.Revision
	}
	data, err := json.Marshal(document)
	if err != nil {
		return value, nil, err
	}
	combined.Files = append(combined.Files, EditorFile{Path: catalogPath, Content: string(data)})
	return combined, revisions, nil
}
func (s *Service) ProcessSavedTests(ctx context.Context) (bool, error) {
	enabled := false
	for _, runner := range s.BundleRunners {
		enabled = enabled || runner == "playwright"
	}
	if !enabled {
		return false, nil
	}
	page, err := s.Store.List(ctx, savedTestQueue(s.Location), "", "", 25)
	if err != nil {
		return false, err
	}
	for _, index := range page.Items {
		request, e := storage.Decode[savedTestRequest](index)
		if e != nil {
			return false, e
		}
		row, e := s.Draft(ctx, request.Actor, index.Project, request.Suite, request.Draft)
		if e == nil && row.Revision == request.Revision {
			combined, revisions, buildErr := s.combinedSavedTests(ctx, request.Actor, index.Project, request.Suite, row)
			if buildErr == nil {
				var data []byte
				data, buildErr = assembleSourceFiles(request.Suite, combined)
				if buildErr == nil {
					var stamp []byte
					stamp, _ = json.Marshal(revisions)
					_, buildErr = s.enqueueBundle(ctx, request.Actor, index.Project, request.Suite, "save-"+digest(string(stamp))[7:], "saved-tests.zip", draftManifestPath(combined), data, nil, &row, revisions, true)
				}
			}
			e = buildErr
		}
		if e != nil {
			message := "The test is saved, but setup could not finish. Review its source and save again."
			if request.Error == message {
				continue
			}
			request.Error = message
			change, e := storage.Put(index.Bucket, index.ID, index.Project, index.Revision, request)
			if e != nil {
				return false, e
			}
			return true, s.Store.Commit(ctx, change)
		}
		if err = s.Store.Commit(ctx, storage.Change{Record: index, Delete: true}); err != nil && !errors.Is(err, storage.ErrConflict) {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
func (s *Service) checkSavedTestJob(ctx context.Context, job Job) error {
	if len(job.SavedDraftRevisions) == 0 {
		return nil
	}
	rows, err := s.savedSourceRows(ctx, job.Project, job.Suite)
	if err != nil {
		return err
	}
	current := map[string]int64{}
	for _, row := range rows {
		value, e := storage.Decode[PlaywrightDraft](row)
		if e != nil {
			return e
		}
		if value.SaveAsTest {
			current[row.ID] = row.Revision
		}
	}
	if !reflect.DeepEqual(current, job.SavedDraftRevisions) {
		return fmt.Errorf("a newer saved test superseded this setup")
	}
	return nil
}
