package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/paging"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) ListTests(ctx context.Context, user, project, routeSuite string, input ListQuery, cursor string) (ListPage, error) {
	result := ListPage{Items: []ListEntry{}}
	if !domain.ValidID(project) || routeSuite != "" && !domain.ValidID(routeSuite) {
		return result, errors.New("valid project and suite identifiers are required")
	}
	if routeSuite != "" {
		if input.Suite != "" && input.Suite != routeSuite {
			return result, errors.New("suite filter must match the route")
		}
		input.Suite = routeSuite
	}
	q, err := normalizeListQuery(input)
	if err != nil {
		return result, err
	}
	object := ""
	if routeSuite != "" {
		object = "suite/" + routeSuite
	}
	if err := s.listAuthority(ctx, user, project, object); err != nil {
		return result, err
	}
	scopeData, _ := json.Marshal(q)
	scope := user + "\x00" + project + "\x00" + routeSuite + "\x00" + string(scopeData)
	raw, err := paging.Open(ctx, s.Store, "catalog-list-settings", scope, cursor)
	if err != nil {
		return result, err
	}
	checkpoint := listCursor{}
	if raw != "" {
		if json.Unmarshal([]byte(raw), &checkpoint) != nil || checkpoint.Offset < 1 || !time.Now().Before(checkpoint.Expires) {
			return result, storage.ErrConflict
		}
	}
	suites, err := s.Store.All(ctx, projects.Bucket("suites", project), project)
	if err != nil {
		return result, err
	}
	all := []listItem{}
	for _, suiteRow := range suites {
		suite, err := storage.Decode[projects.Suite](suiteRow)
		if err != nil {
			return result, err
		}
		if routeSuite != "" && suite.ID != routeSuite {
			continue
		}
		if suite.Archived {
			if routeSuite != "" {
				return result, errors.New("suite is archived")
			}
			continue
		}
		if err := s.Auth.CheckSuiteTestList(ctx, user, project, suite.ID); errors.Is(err, authorization.ErrForbidden) && routeSuite == "" {
			continue
		} else if err != nil {
			return result, err
		}
		rows, err := s.Store.All(ctx, Bucket("tests", project, suite.ID), project)
		if err != nil {
			return result, err
		}
		for _, row := range rows {
			if err := s.Auth.Check(ctx, user, project, "test/"+suite.ID+"/"+row.ID, "catalog.read"); errors.Is(err, authorization.ErrForbidden) {
				continue
			} else if err != nil {
				return result, err
			}
			test, err := storage.Decode[Test](row)
			if err != nil {
				return result, err
			}
			if test.ID != row.ID || suite.ID != suiteRow.ID {
				return result, storage.ErrNotFound
			}
			if q.Application != "" && (test.Application != "" && test.Application != q.Application || test.Application == "" && !slices.Contains(suite.Applications, q.Application)) {
				continue
			}
			all = append(all, listItem{Entry: ListEntry{ID: row.ID, Project: project, Suite: suite.ID, SuiteName: suite.Name, Revision: row.Revision, Data: row.Data, Updated: row.Updated}, Test: test})
		}
	}
	if routeSuite != "" {
		if _, _, err := s.suite(ctx, project, routeSuite); err != nil {
			return result, err
		}
	}
	fingerprint, _ := json.Marshal(all)
	digest := sha256.Sum256(fingerprint)
	snapshot := hex.EncodeToString(digest[:])
	if cursor != "" && checkpoint.Snapshot != snapshot {
		return result, storage.ErrConflict
	}
	result.Total = len(all)
	result.Options, result.Counts = listOptions(all)
	matched, tree := []listItem{}, []listItem{}
	for _, item := range all {
		if q.Suite != "" && item.Entry.Suite != q.Suite {
			continue
		}
		if matchesList(item.Test, item.Entry.SuiteName, q, false) {
			tree = append(tree, item)
		}
		if matchesList(item.Test, item.Entry.SuiteName, q, true) {
			matched = append(matched, item)
		}
	}
	result.Categories = categoryNodes(tree)
	result.Matched = len(matched)
	slices.SortFunc(matched, func(a, b listItem) int { return compareList(a, b, q.Sort) })
	start := checkpoint.Offset
	if start > len(matched) {
		return result, storage.ErrConflict
	}
	end := min(start+q.Limit, len(matched))
	for _, item := range matched[start:end] {
		result.Items = append(result.Items, item.Entry)
	}
	if end < len(matched) {
		data, _ := json.Marshal(listCursor{Offset: end, Snapshot: snapshot, Expires: time.Now().UTC().Add(15 * time.Minute)})
		result.Cursor, err = paging.Seal(ctx, s.Store, "catalog-list-settings", scope, string(data))
	}
	return result, err
}
