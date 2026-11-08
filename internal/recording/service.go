package recording

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store            *storage.Store
	Auth             *authorization.Service
	Driver           Driver
	CompanionEnabled bool
	operation        sync.Mutex
}

func (s *Service) check(ctx context.Context, user, project, suite string, manage bool) error {
	if !domain.ValidID(project) || !domain.ValidID(suite) {
		return errors.New("valid recording project and suite identifiers are required")
	}
	permissions := []string{"sources.read", "catalog.read"}
	if manage {
		permissions = append(permissions, "sources.manage")
	}
	for _, permission := range permissions {
		if err := s.Auth.Check(ctx, user, project, "suite/"+suite, permission); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Available(ctx context.Context, user, project, suite string) (bool, error) {
	if err := s.check(ctx, user, project, suite, false); err != nil {
		return false, err
	}
	return s.Driver != nil || s.CompanionEnabled, nil
}

func (s *Service) Mode() string {
	if s.Driver != nil {
		if _, ok := s.Driver.(InteractiveDriver); ok {
			return "embedded"
		}
		return "local-desktop"
	}
	if s.CompanionEnabled {
		return "companion"
	}
	return "disabled"
}

func (s *Service) List(ctx context.Context, user, project, suite, cursor string, limit int) (storage.Page, error) {
	if err := s.check(ctx, user, project, suite, false); err != nil {
		return storage.Page{}, err
	}
	if limit < 1 || limit > 100 {
		return storage.Page{}, errors.New("recording page limit must be 1 through 100")
	}
	page, err := s.Store.ListNewest(ctx, Bucket(), project, cursor, limit)
	if err != nil {
		return page, err
	}
	items := []storage.Record{}
	for _, row := range page.Items {
		value, err := storage.Decode[Session](row)
		if err != nil {
			return page, err
		}
		if value.Suite == suite && value.Actor == user {
			items = append(items, row)
		}
	}
	page.Items = items
	return page, nil
}

func (s *Service) Start(ctx context.Context, user, project, suite string, input Input) (storage.Record, error) {
	if !domain.ValidID(input.ID) || !domain.ValidID(input.TestID) || !ValidURL(input.URL) || !utf8.ValidString(input.Title) || len(input.Title) < 1 || len(input.Title) > 120 || strings.TrimSpace(input.Title) != input.Title || strings.ContainsFunc(input.Title, unicode.IsControl) {
		return storage.Record{}, errors.New("recording requires permanent identifiers, an HTTP or HTTPS address without credentials, and a title of 1 through 120 characters")
	}
	if s.Driver == nil {
		if s.CompanionEnabled {
			return s.reserve(ctx, user, project, suite, input, "companion")
		}
		return storage.Record{}, errors.New("browser recording requires an enabled local desktop recorder")
	}
	s.operation.Lock()
	defer s.operation.Unlock()
	mode := ""
	if s.Mode() == "embedded" {
		mode = "embedded"
	}
	row, err := s.reserve(ctx, user, project, suite, input, mode)
	if err != nil {
		return row, err
	}
	value, err := storage.Decode[Session](row)
	if err != nil || value.State != "starting" {
		return row, err
	}
	if err := s.Driver.Start(ctx, input.ID, input.URL); err != nil {
		value.State, value.Error = "failed", err.Error()
		if _, saveErr := s.save(ctx, row, value); saveErr != nil {
			return storage.Record{}, saveErr
		}
		return storage.Record{}, err
	}
	value.State = "recording"
	result, err := s.save(context.WithoutCancel(ctx), row, value)
	if err != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		_, _ = s.Driver.Stop(cleanup, input.ID)
	}
	return result, err
}

func (s *Service) owned(ctx context.Context, user, project, suite, id string, manage bool) (Session, storage.Record, error) {
	if err := s.check(ctx, user, project, suite, manage); err != nil {
		return Session{}, storage.Record{}, err
	}
	if !domain.ValidID(id) {
		return Session{}, storage.Record{}, errors.New("invalid recording identifier")
	}
	row, err := s.Store.Get(ctx, Bucket(), id)
	if err != nil {
		return Session{}, row, err
	}
	value, err := storage.Decode[Session](row)
	if err == nil && (value.Actor != user || value.Project != project || value.Suite != suite) {
		err = authorization.ErrForbidden
	}
	return value, row, err
}

func (s *Service) Read(ctx context.Context, user, project, suite, id string) (storage.Record, error) {
	value, row, err := s.owned(ctx, user, project, suite, id, false)
	if err != nil || value.Mode == "companion" || value.State != "recording" {
		return row, err
	}
	if s.Driver == nil {
		return s.finish(ctx, row, value, Snapshot{State: "failed", Error: "The local recording service is unavailable. Start a new recording on an enabled desktop."})
	}
	snapshot, err := s.Driver.Read(ctx, id)
	if err != nil {
		return storage.Record{}, err
	}
	return s.finish(ctx, row, value, snapshot)
}

func (s *Service) Stop(ctx context.Context, user, project, suite, id string, cancel bool) (storage.Record, error) {
	s.operation.Lock()
	defer s.operation.Unlock()
	value, row, err := s.owned(ctx, user, project, suite, id, true)
	if err != nil || !active(value.State) {
		return row, err
	}
	if value.Mode == "companion" {
		return s.stopCompanion(ctx, user, project, suite, id, cancel)
	}
	if s.Driver == nil {
		return storage.Record{}, errors.New("the local recording service is unavailable")
	}
	snapshot, err := s.Driver.Stop(ctx, id)
	if err != nil {
		return storage.Record{}, err
	}
	if cancel {
		snapshot.State = "cancelled"
	}
	return s.finish(ctx, row, value, snapshot)
}

func (s *Service) finish(ctx context.Context, row storage.Record, value Session, snapshot Snapshot) (storage.Record, error) {
	if snapshot.State == "failed" && snapshot.Code == "" {
		snapshot.Code = value.Code
		snapshot.Steps = value.Steps
	}
	if len(snapshot.Code) > ContentLimit || !utf8.ValidString(snapshot.Code) {
		return storage.Record{}, errors.New("recorded code exceeds its UTF-8 text limit")
	}
	if snapshot.State != "recording" && snapshot.State != "ready" && snapshot.State != "cancelled" && snapshot.State != "failed" {
		return storage.Record{}, errors.New("invalid recorder state")
	}
	if len(snapshot.Steps) > 500 {
		return storage.Record{}, errors.New("recording exceeds its 500 step limit")
	}
	if value.State == snapshot.State && value.Code == snapshot.Code && value.Error == snapshot.Error && reflect.DeepEqual(value.Steps, snapshot.Steps) {
		return row, nil
	}
	value.State, value.Code, value.Error = snapshot.State, snapshot.Code, snapshot.Error
	value.Steps = snapshot.Steps
	result, err := s.save(ctx, row, value)
	if errors.Is(err, storage.ErrConflict) {
		return s.Store.Get(ctx, row.Bucket, row.ID)
	}
	return result, err
}

func (s *Service) save(ctx context.Context, row storage.Record, value Session) (storage.Record, error) {
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err != nil {
		return storage.Record{}, err
	}
	if err := s.Store.Commit(ctx, change); err != nil {
		return storage.Record{}, err
	}
	return s.Store.Get(ctx, row.Bucket, row.ID)
}
