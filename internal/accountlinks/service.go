package accountlinks

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/localauth"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
)

var ErrInvalid = errors.New("identity link is unavailable, expired, or no longer valid")
var ErrClaimed = errors.New("this identity is already linked; accounts cannot be merged")

const bucket = "identity-link-operations"

type providerState struct {
	ID, Name, Issuer string
	Disabled         bool
}

type Provider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Issuer   string `json:"issuer"`
	Revision int64  `json:"revision"`
}

type Proof struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
}

type View struct {
	ID             string    `json:"id"`
	Revision       int64     `json:"revision"`
	State          string    `json:"state"`
	SourceMethod   string    `json:"sourceMethod"`
	SourceProvider *Provider `json:"sourceProvider,omitempty"`
	TargetProvider Provider  `json:"targetProvider"`
	Identity       *Proof    `json:"identity,omitempty"`
	Reason         string    `json:"reason"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

type operation struct {
	View
	UserID            string    `json:"userId"`
	SessionID         string    `json:"sessionId"`
	ActiveTransaction string    `json:"activeTransaction,omitempty"`
	SourceVerifiedAt  time.Time `json:"sourceVerifiedAt"`
	TargetVerifiedAt  time.Time `json:"targetVerifiedAt"`
	ProofStartedAt    time.Time `json:"proofStartedAt"`
}

type Service struct {
	Store         *storage.Store
	Sessions      *sessions.Service
	LocalAccounts *localauth.Service
	LocalUser     string
}

type StartInput struct {
	ProviderID   string `json:"providerId"`
	Reason       string `json:"reason"`
	Password     string `json:"password,omitempty"`
	ConfirmLocal bool   `json:"confirmLocal,omitempty"`
}

type Options struct {
	SourceMethod string     `json:"sourceMethod"`
	CanStart     bool       `json:"canStart"`
	Providers    []Provider `json:"providers"`
}

func (s *Service) provider(ctx context.Context, id string) (Provider, error) {
	row, err := s.Store.Get(ctx, "providers", id)
	if err != nil {
		return Provider{}, err
	}
	p, err := storage.Decode[providerState](row)
	if err != nil {
		return Provider{}, err
	}
	if row.Project != "" || p.ID != id || p.Disabled || p.Issuer == "" || row.Revision < 1 {
		return Provider{}, ErrInvalid
	}
	return Provider{ID: p.ID, Name: p.Name, Issuer: p.Issuer, Revision: row.Revision}, nil
}

func (s *Service) Options(ctx context.Context, sessionID, peer string) (Options, error) {
	active, err := s.Sessions.Validate(ctx, sessionID)
	if err != nil {
		return Options{}, err
	}
	result := Options{SourceMethod: active.AuthenticationMethod, Providers: []Provider{}}
	result.CanStart = !active.MustChangePassword && (active.AuthenticationMethod == "oidc" || (active.AuthenticationMethod == "password" && s.LocalAccounts != nil) || (active.AuthenticationMethod == "local" && active.UserID == s.LocalUser && loopback(peer)))
	rows, err := s.Store.All(ctx, "providers", "")
	if err != nil {
		return Options{}, err
	}
	for _, row := range rows {
		p, err := s.provider(ctx, row.ID)
		if errors.Is(err, ErrInvalid) {
			continue
		}
		if err != nil {
			return Options{}, err
		}
		result.Providers = append(result.Providers, p)
	}
	return result, nil
}

func loopback(peer string) bool { ip := net.ParseIP(peer); return ip != nil && ip.IsLoopback() }

func (s *Service) transaction(ctx context.Context, fn func(*Service) error) error {
	version, err := s.Sessions.Directory.Revision(ctx)
	if err != nil {
		return err
	}
	return s.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
		directory := &identity.Directory{Store: store}
		if err := directory.CommitAt(ctx, version); err != nil {
			return err
		}
		bound := *s
		bound.Store = store
		boundSessions := *s.Sessions
		boundSessions.Store, boundSessions.Directory = store, directory
		bound.Sessions = &boundSessions
		return fn(&bound)
	})
}

func (s *Service) Start(ctx context.Context, sessionID string, input StartInput, peer string) (View, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if !domain.ValidID(input.ProviderID) || input.Reason == "" || len(input.Reason) > 1024 || !utf8.ValidString(input.Reason) {
		return View{}, errors.New("provider and a bounded reason are required")
	}
	active, err := s.Sessions.Validate(ctx, sessionID)
	if err != nil {
		return View{}, err
	}
	if active.MustChangePassword {
		return View{}, sessions.ErrPasswordChangeRequired
	}
	switch active.AuthenticationMethod {
	case "password":
		if s.LocalAccounts == nil {
			return View{}, ErrInvalid
		}
		if err := s.LocalAccounts.VerifyCurrent(ctx, active, input.Password, peer); err != nil {
			return View{}, err
		}
	case "local":
		if !input.ConfirmLocal || active.UserID != s.LocalUser || !loopback(peer) {
			return View{}, ErrInvalid
		}
	case "oidc":
	default:
		return View{}, ErrInvalid
	}
	var result View
	err = s.transaction(ctx, func(bound *Service) error {
		current, err := bound.Sessions.Validate(ctx, sessionID)
		if err != nil {
			return err
		}
		if current.UserID != active.UserID || current.UserEpoch != active.UserEpoch || current.AuthenticationMethod != active.AuthenticationMethod || current.MustChangePassword {
			return ErrInvalid
		}
		target, err := bound.provider(ctx, input.ProviderID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		until := now.Add(5 * time.Minute)
		if current.AuthenticationUntil.Before(until) {
			until = current.AuthenticationUntil
		}
		if current.ExpiresAt.Before(until) {
			until = current.ExpiresAt
		}
		op := operation{View: View{ID: domain.NewID(), Revision: 1, State: "prove-target", SourceMethod: current.AuthenticationMethod, TargetProvider: target, Reason: input.Reason, ExpiresAt: until}, UserID: current.UserID, SessionID: current.ID, SourceVerifiedAt: now}
		if current.AuthenticationMethod == "oidc" {
			source, err := bound.provider(ctx, current.Provider)
			if err != nil {
				return err
			}
			if source.Revision != current.ProviderRevision {
				return ErrInvalid
			}
			op.SourceProvider, op.SourceVerifiedAt, op.State = &source, time.Time{}, "authenticate-source"
		}
		change, err := storage.Put(bucket, op.ID, "", 0, op)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(current.UserID, "", "identity.link.start", op.ID, nil, map[string]string{"providerId": target.ID, "reason": op.Reason})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change, audit); err != nil {
			return err
		}
		result = op.View
		return nil
	})
	return result, err
}

func (s *Service) load(ctx context.Context, sessionID, id string) (storage.Record, operation, error) {
	active, err := s.Sessions.Validate(ctx, sessionID)
	if err != nil {
		return storage.Record{}, operation{}, err
	}
	row, err := s.Store.Get(ctx, bucket, id)
	if err != nil {
		return row, operation{}, err
	}
	op, err := storage.Decode[operation](row)
	if err != nil {
		return row, op, err
	}
	if row.Project != "" || op.ID != id || op.SessionID != active.ID || op.UserID != active.UserID {
		return storage.Record{}, operation{}, storage.ErrNotFound
	}
	op.Revision = row.Revision
	return row, op, nil
}

func terminal(state string) bool {
	return state == "complete" || state == "cancelled" || state == "failed"
}

func (s *Service) current(ctx context.Context, op operation) error {
	if terminal(op.State) || !time.Now().Before(op.ExpiresAt) {
		return ErrInvalid
	}
	for _, expected := range []*Provider{&op.TargetProvider, op.SourceProvider} {
		if expected == nil {
			continue
		}
		p, err := s.provider(ctx, expected.ID)
		if err != nil {
			return err
		}
		if p.Revision != expected.Revision || p.Issuer != expected.Issuer {
			return ErrInvalid
		}
	}
	return nil
}

func (s *Service) Read(ctx context.Context, sessionID, id string) (View, error) {
	_, op, err := s.load(ctx, sessionID, id)
	if err != nil {
		return View{}, err
	}
	if !terminal(op.State) {
		if err := s.current(ctx, op); err != nil {
			if !errors.Is(err, ErrInvalid) && !errors.Is(err, storage.ErrNotFound) {
				return View{}, err
			}
			op.State, op.Identity = "expired", nil
		}
	}
	return op.View, nil
}

func (s *Service) Cancel(ctx context.Context, sessionID, id string, revision int64) (View, error) {
	var result View
	err := s.transaction(ctx, func(bound *Service) error {
		row, op, err := bound.load(ctx, sessionID, id)
		if err != nil {
			return err
		}
		if revision < 1 || row.Revision != revision {
			return storage.ErrConflict
		}
		if terminal(op.State) {
			return ErrInvalid
		}
		op.State, op.Identity, op.ActiveTransaction = "cancelled", nil, ""
		op.Revision++
		change, err := storage.Put(bucket, id, "", row.Revision, op)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(op.UserID, "", "identity.link.cancel", op.ID, nil, map[string]string{"reason": op.Reason})
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change, audit); err != nil {
			return err
		}
		result = op.View
		return nil
	})
	return result, err
}
