package cliauth

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Sessions *sessions.Service
	Now      func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
func (s *Service) Options() Options {
	return Options{ServerTime: s.now(), LifetimeSeconds: int(Lifetime / time.Second), PollSeconds: int(PollInterval / time.Second)}
}
func (s *Service) Read(ctx context.Context, parent, id, client string) (View, error) {
	if _, err := s.Sessions.Validate(ctx, parent); err != nil {
		return View{}, err
	}
	until, err := expires(id, s.now())
	if err != nil || !ValidClient(client) {
		return View{}, ErrInvalid
	}
	row, err := s.Sessions.Store.Get(ctx, "cli-authorizations", id)
	if errors.Is(err, storage.ErrNotFound) {
		return View{ID: id, Client: client, State: "pending", ExpiresAt: until}, nil
	}
	if err != nil {
		return View{}, err
	}
	current, err := storage.Decode[approval](row)
	if err != nil {
		return View{}, err
	}
	if current.Client != client {
		return View{}, ErrInvalid
	}
	return current.View, nil
}
func (s *Service) Confirm(ctx context.Context, parent, id string, input Confirmation) (View, error) {
	current, err := s.Sessions.Validate(ctx, parent)
	if err != nil {
		return View{}, err
	}
	if current.MustChangePassword {
		return View{}, sessions.ErrPasswordChangeRequired
	}
	view, err := s.Read(ctx, parent, id, input.Client)
	if err != nil {
		return View{}, err
	}
	if input.Approve && !codeMatches(id, input.Code) {
		return View{}, errors.New("terminal code does not match this CLI sign-in")
	}
	existing, err := s.Sessions.Store.Get(ctx, "cli-authorizations", id)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return View{}, err
	}
	if existing.Revision > 0 {
		previous, err := storage.Decode[approval](existing)
		if err != nil {
			return View{}, err
		}
		if previous.Parent == parent && ((input.Approve && previous.State == "approved") || (!input.Approve && previous.State == "denied")) {
			return previous.View, nil
		}
		return View{}, storage.ErrConflict
	}
	view.State = "denied"
	if input.Approve {
		view.State = "approved"
	}
	change, err := storage.Put("cli-authorizations", id, "", 0, approval{View: view, Parent: parent})
	if err != nil {
		return View{}, err
	}
	original, err := s.Sessions.Store.Get(ctx, "sessions", parent)
	if err != nil {
		return View{}, err
	}
	audit, err := domain.Audit(current.UserID, "", "session.cli."+view.State, id, nil, map[string]any{"clientName": input.Client, "expiresAt": view.ExpiresAt})
	if err != nil {
		return View{}, err
	}
	if err := s.Sessions.Store.Commit(ctx, change, storage.Change{Record: original, Assert: true}, audit); err != nil {
		return View{}, err
	}
	return view, nil
}
func (s *Service) Exchange(ctx context.Context, input Proof) (Exchange, error) {
	if err := verify(input, s.now()); err != nil {
		return Exchange{}, err
	}
	row, err := s.Sessions.Store.Get(ctx, "cli-authorizations", input.Request)
	if errors.Is(err, storage.ErrNotFound) {
		return Exchange{State: "pending"}, nil
	}
	if err != nil {
		return Exchange{}, err
	}
	approved, err := storage.Decode[approval](row)
	if err != nil {
		return Exchange{}, err
	}
	if approved.Client != input.Client {
		return Exchange{}, ErrInvalid
	}
	if terminal(approved.State) {
		return Exchange{State: approved.State}, nil
	}
	if approved.State != "approved" {
		return Exchange{}, ErrInvalid
	}
	session, cookie, err := s.Sessions.Delegate(ctx, approved.Parent, approved.Client, func(next sessions.Session) ([]storage.Change, error) {
		approved.State, approved.Session = "consumed", next.ID
		change, err := storage.Put(row.Bucket, row.ID, "", row.Revision, approved)
		if err != nil {
			return nil, err
		}
		return []storage.Change{change}, nil
	})
	if errors.Is(err, storage.ErrConflict) {
		return Exchange{}, storage.ErrConflict
	}
	if err != nil {
		return Exchange{}, err
	}
	return Exchange{State: "authorized", Session: &Grant{Cookie: cookie.Value, CSRF: session.CSRF, ExpiresAt: session.ExpiresAt}}, nil
}
func (s *Service) Cancel(ctx context.Context, input Proof) error {
	if err := verify(input, s.now()); err != nil {
		return err
	}
	row, err := s.Sessions.Store.Get(ctx, "cli-authorizations", input.Request)
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	approved, err := storage.Decode[approval](row)
	if err != nil {
		return err
	}
	if approved.Client != input.Client {
		return ErrInvalid
	}
	if approved.State == "denied" || approved.State == "cancelled" {
		return nil
	}
	changes := []storage.Change{}
	if approved.Session != "" {
		session, err := s.Sessions.Store.Get(ctx, "sessions", approved.Session)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if session.Revision > 0 {
			changes = append(changes, storage.Change{Record: session, Delete: true})
		}
	}
	approved.State = "cancelled"
	change, err := storage.Put(row.Bucket, row.ID, "", row.Revision, approved)
	if err != nil {
		return err
	}
	return s.Sessions.Store.Commit(ctx, append(changes, change)...)
}
func (s *Service) Sweep(ctx context.Context) error {
	page, err := s.Sessions.Store.List(ctx, "cli-authorizations", "", "", 100)
	if err != nil {
		return err
	}
	changes := []storage.Change{}
	for _, row := range page.Items {
		value, err := storage.Decode[approval](row)
		if err != nil {
			return err
		}
		if !s.now().Before(value.ExpiresAt) {
			changes = append(changes, storage.Change{Record: row, Delete: true})
		} else {
			break
		}
	}
	if len(changes) == 0 {
		return nil
	}
	return s.Sessions.Store.Commit(ctx, changes...)
}
func (s *Service) StartMaintenance(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTicker(time.Minute)
		defer timer.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			_ = s.Sweep(ctx)
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
	}()
	return done
}
