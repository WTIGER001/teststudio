package accountlinks

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/domain"
	"time"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/storage"
)

type ProofRequest struct {
	OperationID      string    `json:"operationId"`
	Revision         int64     `json:"revision"`
	Step             string    `json:"step"`
	ProviderID       string    `json:"providerId"`
	ProviderRevision int64     `json:"providerRevision"`
	StartedAt        time.Time `json:"startedAt"`
	TransactionID    string    `json:"transactionId"`
}

func proofProvider(op operation) (*Provider, error) {
	if op.State == "authenticate-source" && op.SourceProvider != nil {
		return op.SourceProvider, nil
	}
	if op.State == "prove-target" {
		return &op.TargetProvider, nil
	}
	return nil, ErrInvalid
}

func (s *Service) PrepareProof(ctx context.Context, sessionID, id, transactionID string) (ProofRequest, error) {
	row, op, err := s.load(ctx, sessionID, id)
	if err != nil {
		return ProofRequest{}, err
	}
	if err := s.current(ctx, op); err != nil {
		return ProofRequest{}, err
	}
	p, err := proofProvider(op)
	if err != nil {
		return ProofRequest{}, err
	}
	return ProofRequest{OperationID: id, Revision: row.Revision + 1, Step: op.State, ProviderID: p.ID, ProviderRevision: p.Revision, StartedAt: time.Now().UTC(), TransactionID: transactionID}, nil
}

func (s *Service) BindProof(ctx context.Context, sessionID string, request ProofRequest, login storage.Change) error {
	return s.transaction(ctx, func(bound *Service) error {
		row, op, err := bound.load(ctx, sessionID, request.OperationID)
		if err != nil {
			return err
		}
		if err := bound.current(ctx, op); err != nil {
			return err
		}
		p, err := proofProvider(op)
		if err != nil {
			return err
		}
		if request.Revision != row.Revision+1 || request.Step != op.State || request.ProviderID != p.ID || request.ProviderRevision != p.Revision || len(request.TransactionID) != 64 || login.Record.Bucket != "login-transactions" || login.Record.ID != request.TransactionID || login.Record.Revision != 0 || login.Delete || login.Assert {
			return ErrInvalid
		}
		op.ActiveTransaction, op.ProofStartedAt, op.Revision = request.TransactionID, request.StartedAt, request.Revision
		change, err := storage.Put(bucket, op.ID, "", row.Revision, op)
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, login)
	})
}

func (s *Service) proofOperation(ctx context.Context, sessionID string, request ProofRequest) (storage.Record, operation, error) {
	row, op, err := s.load(ctx, sessionID, request.OperationID)
	if err != nil {
		return row, op, err
	}
	if err := s.current(ctx, op); err != nil {
		return row, op, err
	}
	p, err := proofProvider(op)
	if err != nil {
		return row, op, err
	}
	if row.Revision != request.Revision || op.State != request.Step || p.ID != request.ProviderID || p.Revision != request.ProviderRevision || op.ActiveTransaction == "" || op.ActiveTransaction != request.TransactionID || !op.ProofStartedAt.Equal(request.StartedAt) {
		return row, op, ErrInvalid
	}
	return row, op, nil
}

func FreshAuthentication(authenticatedAt, startedAt, now time.Time) bool {
	return !authenticatedAt.IsZero() && !startedAt.IsZero() && !authenticatedAt.Before(startedAt.Add(-time.Minute)) && !authenticatedAt.After(now.Add(time.Minute)) && now.Sub(authenticatedAt) <= 5*time.Minute
}

func (s *Service) AcceptProof(ctx context.Context, sessionID string, request ProofRequest, proof Proof, authenticatedAt, expiresAt time.Time) (View, error) {
	var result View
	err := s.transaction(ctx, func(bound *Service) error {
		row, op, err := bound.proofOperation(ctx, sessionID, request)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if !FreshAuthentication(authenticatedAt, request.StartedAt, now) || !now.Before(expiresAt) || proof.Subject == "" || len(proof.Subject) > 255 || !utf8.ValidString(proof.Subject) {
			return ErrInvalid
		}
		p, _ := proofProvider(op)
		if proof.Issuer != p.Issuer {
			return ErrInvalid
		}
		link, err := bound.Sessions.Directory.ExternalLink(ctx, proof.Issuer, proof.Subject)
		if op.State == "authenticate-source" {
			if err != nil || link.UserID != op.UserID {
				return ErrInvalid
			}
			op.SourceVerifiedAt, op.State = now, "prove-target"
		} else {
			if err == nil {
				return ErrClaimed
			}
			if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
			op.Identity, op.TargetVerifiedAt, op.State = &proof, now, "review"
		}
		if expiresAt.Before(op.ExpiresAt) {
			op.ExpiresAt = expiresAt
		}
		op.ActiveTransaction, op.ProofStartedAt, op.Revision = "", time.Time{}, row.Revision+1
		change, err := storage.Put(bucket, op.ID, "", row.Revision, op)
		if err != nil {
			return err
		}
		if err := bound.Store.Commit(ctx, change); err != nil {
			return err
		}
		result = op.View
		return nil
	})
	return result, err
}

func (s *Service) RejectProof(ctx context.Context, sessionID string, request ProofRequest) error {
	return s.transaction(ctx, func(bound *Service) error {
		row, op, err := bound.load(ctx, sessionID, request.OperationID)
		if err != nil {
			return err
		}
		if row.Revision != request.Revision || op.ActiveTransaction != request.TransactionID || op.ActiveTransaction == "" || terminal(op.State) {
			return ErrInvalid
		}
		op.State, op.Identity, op.ActiveTransaction, op.ProofStartedAt, op.Revision = "failed", nil, "", time.Time{}, row.Revision+1
		change, err := storage.Put(bucket, op.ID, "", row.Revision, op)
		if err != nil {
			return err
		}
		audit, err := domain.Audit(op.UserID, "", "identity.link.reject", op.ID, nil, map[string]string{"reason": op.Reason, "providerId": op.TargetProvider.ID})
		if err != nil {
			return err
		}
		return bound.Store.Commit(ctx, change, audit)
	})
}

func (s *Service) CheckProof(ctx context.Context, sessionID string, request ProofRequest) error {
	_, _, err := s.proofOperation(ctx, sessionID, request)
	return err
}
