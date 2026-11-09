package statuspages

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/probes"
	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store  *storage.Store
	Auth   *authorization.Service
	Probes *probes.Service
}

func Bucket(project string) string { return "status-pages/" + project }
func publicToken() string {
	var bytes [20]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes[:])
}

var color = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (s *Service) authority(ctx context.Context, user, project, id, action string) error {
	if !domain.ValidID(project) || !domain.ValidID(id) {
		return storage.ErrNotFound
	}
	if _, err := s.Store.Get(ctx, "projects", project); err != nil {
		return err
	}
	if err := s.Auth.Check(ctx, user, project, "", "statuspages."+action); err != nil {
		return err
	}
	return s.Auth.Check(ctx, user, project, "status-page/"+id, "statuspages."+action)
}
func (s *Service) Read(ctx context.Context, user, project, id string) (Page, storage.Record, error) {
	if err := s.authority(ctx, user, project, id, "read"); err != nil {
		return Page{}, storage.Record{}, err
	}
	row, err := s.Store.Get(ctx, Bucket(project), id)
	if err != nil {
		return Page{}, row, err
	}
	page, err := storage.Decode[Page](row)
	return page, row, err
}
func (s *Service) List(ctx context.Context, user, project, cursor string) (storage.Page, error) {
	if err := s.authority(ctx, user, project, "list", "read"); err != nil {
		return storage.Page{}, err
	}
	page, err := s.Store.List(ctx, Bucket(project), project, cursor, 30)
	if err != nil {
		return page, err
	}
	rows := []storage.Record{}
	for _, row := range page.Items {
		if err := s.authority(ctx, user, project, row.ID, "read"); err == nil {
			rows = append(rows, row)
		} else if !errors.Is(err, authorization.ErrForbidden) {
			return page, err
		}
	}
	page.Items = rows
	return page, nil
}
func (s *Service) validate(ctx context.Context, user, project string, d Definition) ([]storage.Change, error) {
	if !domain.ValidID(d.ID) || strings.TrimSpace(d.Title) == "" || len(d.Title) > 256 || !color.MatchString(d.Background) || d.MaxAge < 60 || d.MaxAge > 86400 || len(d.Probes) < 1 || len(d.Probes) > 100 {
		return nil, errors.New("a title, hex background, freshness 60–86400 seconds, and 1–100 probes are required")
	}
	if _, err := WindowStart(d.Window, time.Now()); err != nil {
		return nil, err
	}
	if err := validateLogo(d.Logo); err != nil {
		return nil, err
	}
	if d.LogoHeight < 0 || d.LogoHeight > 1024 {
		return nil, errors.New("logo height must be a whole number from 1 through 1024 pixels, or 0 for automatic sizing")
	}
	if len(d.LogoAlt) > 256 {
		return nil, errors.New("logo alternative text must be at most 256 bytes")
	}
	seen := map[string]bool{}
	changes := []storage.Change{}
	for _, entry := range d.Probes {
		key := entry.Probe
		if key == "" {
			key = entry.Suite + "/" + entry.Test + "/" + entry.Environment + "/" + entry.Location
		}
		if seen[key] || strings.TrimSpace(entry.Title) == "" || len(entry.Title) > 256 || len(entry.Description) > 2048 {
			return nil, errors.New("choose unique probes with public titles up to 256 bytes and descriptions up to 2048 bytes")
		}
		seen[key] = true
		if entry.Probe == "" {
			_, _, _, disabled, err := s.resolve(ctx, user, project, entry)
			if err != nil {
				return nil, err
			}
			if disabled {
				return nil, errors.New("select a current enabled check")
			}
			if err = s.Auth.Check(ctx, user, project, "", "runs.read"); err != nil {
				return nil, err
			}
			continue
		}
		if entry.Suite != "" || entry.Test != "" || entry.Environment != "" || entry.Location != "" {
			return nil, errors.New("choose either a probe or a direct test reference")
		}
		binding, row, err := s.Probes.Read(ctx, user, project, entry.Probe)
		if err != nil {
			return nil, err
		}
		if binding.Archived {
			return nil, errors.New("restore the selected probe before including it")
		}
		test, err := s.Probes.Catalog.RetainedTest(ctx, user, project, binding.Suite, binding.Test)
		if err != nil {
			return nil, err
		}
		if !test.Current || test.SuiteArchived || test.Test.Type != "https" || test.Test.Runner != "http" || test.Test.Check == nil {
			return nil, errors.New("select a current native HTTP probe")
		}
		if err = s.Auth.Check(ctx, user, project, "", "runs.read"); err != nil {
			return nil, err
		}
		changes = append(changes, storage.Change{Record: row, Assert: true})
	}
	return changes, nil
}
func (s *Service) Save(ctx context.Context, user, project string, d Definition, revision int64) (storage.Record, error) {
	ctx, err := s.Auth.BeginMutation(ctx)
	if err != nil {
		return storage.Record{}, err
	}
	d.Logo = strings.TrimSpace(d.Logo)
	d.LogoAlt = strings.TrimSpace(d.LogoAlt)
	if d.Logo == "" {
		d.LogoAlt = ""
		d.LogoHeight = 0
	}
	d.Slug, err = NormalizeSlug(d.Slug)
	if err != nil {
		return storage.Record{}, err
	}
	if err = s.authority(ctx, user, project, d.ID, "manage"); err != nil {
		return storage.Record{}, err
	}
	assertions, err := s.validate(ctx, user, project, d)
	if err != nil {
		return storage.Record{}, err
	}
	page := Page{Definition: d, Token: publicToken()}
	before, err := s.Store.Get(ctx, Bucket(project), d.ID)
	if err == nil {
		if revision == 0 {
			return storage.Record{}, storage.ErrConflict
		}
		page, err = storage.Decode[Page](before)
		if err != nil {
			return before, err
		}
		page.Definition = d
	} else if !errors.Is(err, storage.ErrNotFound) || revision != 0 {
		return storage.Record{}, err
	}
	if d.Enabled != nil {
		if *d.Enabled || page.Published != nil {
			if err := s.authority(ctx, user, project, d.ID, "publish"); err != nil {
				return storage.Record{}, err
			}
		}
		if *d.Enabled {
			now := time.Now().UTC()
			snapshot := d
			page.Published, page.Publisher, page.PublishedAt = &snapshot, user, &now
		} else {
			page.Published, page.Publisher, page.PublishedAt = nil, "", nil
		}
		token, err := s.Store.Get(ctx, "public-status-pages", page.Token)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.Record{}, err
		}
		link, err := storage.Put("public-status-pages", page.Token, project, token.Revision, map[string]string{"pageId": d.ID})
		if err != nil {
			return storage.Record{}, err
		}
		assertions = append(assertions, link)
	}
	reservation, err := s.slugReservation(ctx, project, page)
	if err != nil {
		return storage.Record{}, err
	}
	assertions = append(assertions, reservation...)
	change, err := storage.Put(Bucket(project), d.ID, project, revision, page)
	if err != nil {
		return storage.Record{}, err
	}
	audit, err := domain.Audit(user, project, "status-page.save", d.ID, before.Data, d)
	if err != nil {
		return storage.Record{}, err
	}
	assertions = append(assertions, authorization.AuthorityAssertions(ctx)...)
	projectRow, err := s.Store.Get(ctx, "projects", project)
	if err != nil {
		return storage.Record{}, err
	}
	p, _ := storage.Decode[projects.Project](projectRow)
	if p.Archived {
		return storage.Record{}, errors.New("restore the project before editing status pages")
	}
	assertions = append(assertions, storage.Change{Record: projectRow, Assert: true}, change, audit)
	if err = s.Store.CommitAll(ctx, assertions...); err != nil {
		if errors.Is(err, storage.ErrConflict) && page.Slug != "" {
			row, e := s.Store.Get(ctx, "status-page-slugs", page.Slug)
			if e == nil && !ownsSlug(row, project, page.ID, page.Token) {
				return storage.Record{}, ErrSlugTaken
			}
		}
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, Bucket(project), d.ID)
}
func (s *Service) Publish(ctx context.Context, user, project, id string, revision int64, publish bool) (storage.Record, error) {
	ctx, err := s.Auth.BeginMutation(ctx)
	if err != nil {
		return storage.Record{}, err
	}
	if err = s.authority(ctx, user, project, id, "publish"); err != nil {
		return storage.Record{}, err
	}
	page, before, err := s.Read(ctx, user, project, id)
	if err != nil {
		return before, err
	}
	if before.Revision != revision {
		return storage.Record{}, storage.ErrConflict
	}
	assertions := []storage.Change{}
	if publish {
		assertions, err = s.validate(ctx, user, project, page.Definition)
		if err != nil {
			return storage.Record{}, err
		}
		reservation, e := s.slugReservation(ctx, project, page)
		if e != nil {
			return before, e
		}
		assertions = append(assertions, reservation...)
		now := time.Now().UTC()
		snapshot := page.Definition
		page.Published = &snapshot
		page.Publisher = user
		page.PublishedAt = &now
	} else {
		page.Published = nil
		page.Publisher = ""
		page.PublishedAt = nil
	}
	change, err := storage.Put(Bucket(project), id, project, revision, page)
	if err != nil {
		return before, err
	}
	token, err := s.Store.Get(ctx, "public-status-pages", page.Token)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return before, err
	}
	link, err := storage.Put("public-status-pages", page.Token, project, token.Revision, map[string]string{"pageId": id})
	if err != nil {
		return before, err
	}
	audit, err := domain.Audit(user, project, "status-page.publish", id, before.Data, page)
	if err != nil {
		return before, err
	}
	assertions = append(assertions, authorization.AuthorityAssertions(ctx)...)
	assertions = append(assertions, change, link, audit)
	if err = s.Store.CommitAll(ctx, assertions...); err != nil {
		return before, err
	}
	return s.Store.Get(ctx, Bucket(project), id)
}
func (s *Service) Delete(ctx context.Context, user, project, id string, revision int64) error {
	ctx, err := s.Auth.BeginMutation(ctx)
	if err != nil {
		return err
	}
	if err = s.authority(ctx, user, project, id, "delete"); err != nil {
		return err
	}
	_, row, err := s.Read(ctx, user, project, id)
	if err != nil {
		return err
	}
	if row.Revision != revision {
		return storage.ErrConflict
	}
	audit, err := domain.Audit(user, project, "status-page.delete", id, row.Data, nil)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, append(authorization.AuthorityAssertions(ctx), storage.Change{Record: row, Delete: true}, audit)...)
}
