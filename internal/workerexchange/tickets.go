package workerexchange

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func fingerprint(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func (s *Service) key(ctx context.Context, create bool) ([]byte, error) {
	for range 8 {
		row, err := s.Store.Get(ctx, "runtime-private", "worker-exchange-key")
		if err == nil {
			key, err := storage.Decode[[]byte](row)
			if err != nil || len(key) != 32 {
				return nil, ErrUnavailable
			}
			return key, nil
		}
		if !create || !errors.Is(err, storage.ErrNotFound) {
			return nil, ErrUnavailable
		}
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, ErrUnavailable
		}
		change, err := storage.Put("runtime-private", "worker-exchange-key", "", 0, key)
		if err != nil {
			return nil, ErrUnavailable
		}
		if err := s.Store.Commit(ctx, change); errors.Is(err, storage.ErrConflict) {
			continue
		} else if err != nil {
			return nil, ErrUnavailable
		}
		return key, nil
	}
	return nil, ErrConflict
}

func (s *Service) token(ctx context.Context, record Record) (string, error) {
	key, err := s.key(ctx, false)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("teststudio.worker/v1\x00" + record.Reference.Installation + "\x00" + record.Reference.ID + "\x00" + record.Nonce))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) Prepare(ctx context.Context, ref execution.Reference, input execution.CodedInput, deadline time.Time) (Ticket, error) {
	if !domain.ValidID(ref.ID) || !domain.ValidID(ref.Installation) || !domain.ValidID(ref.Project) || !domain.ValidID(ref.Run) || !domain.ValidID(ref.Test) || !domain.ValidID(ref.Worker) || ref.Attempt < 1 || ref.Profile == "" || input.RunID != ref.Run || input.Test.ID != ref.Test || input.Artifact.SourceProject != "" && input.Artifact.SourceProject != ref.Project {
		return Ticket{}, ErrInvalid
	}
	payload, err := execution.MarshalWorkItem(input)
	if err != nil {
		return Ticket{}, ErrInvalid
	}
	if s.Authorize == nil {
		return Ticket{}, ErrDenied
	}
	for range 8 {
		access, err := s.Authorize(ctx, ref)
		if err != nil {
			return Ticket{}, ErrDenied
		}
		if access.Deadline.Before(deadline) {
			deadline = access.Deadline
		}
		if !deadline.After(time.Now()) || deadline.After(time.Now().Add(2*time.Hour)) {
			return Ticket{}, ErrInvalid
		}
		row, err := s.Store.Get(ctx, Bucket, ref.ID)
		var record Record
		if err == nil {
			record, err = storage.Decode[Record](row)
			if err != nil {
				return Ticket{}, ErrUnavailable
			}
			if record.Reference != ref || record.InputHash != fingerprint(payload) || !record.Deadline.Equal(deadline) || record.Revoked {
				return Ticket{}, ErrConflict
			}
		} else if errors.Is(err, storage.ErrNotFound) {
			if _, err := s.key(ctx, true); err != nil {
				return Ticket{}, err
			}
			nonce := make([]byte, 16)
			if _, err := rand.Read(nonce); err != nil {
				return Ticket{}, ErrUnavailable
			}
			record = Record{Reference: ref, Input: payload, InputHash: fingerprint(payload), Nonce: hex.EncodeToString(nonce), CreatedAt: time.Now().UTC(), Deadline: deadline, Uploaded: map[string]bool{}}
			change, err := storage.Put(Bucket, ref.ID, ref.Project, 0, record)
			if err != nil {
				return Ticket{}, ErrUnavailable
			}
			changes := []storage.Change{change}
			for _, guard := range access.Records {
				changes = append(changes, storage.Change{Record: guard, Assert: true})
			}
			if err := s.Store.Commit(ctx, changes...); errors.Is(err, storage.ErrConflict) {
				continue
			} else if err != nil {
				return Ticket{}, ErrUnavailable
			}
		} else {
			return Ticket{}, ErrUnavailable
		}
		token, err := s.token(ctx, record)
		if err != nil {
			return Ticket{}, err
		}
		return Ticket{Execution: ref.ID, Token: token, Deadline: record.Deadline}, nil
	}
	return Ticket{}, ErrConflict
}

func (s *Service) authenticated(ctx context.Context, id, token, instance string) (Record, storage.Record, []storage.Record, error) {
	if !domain.ValidID(id) || !domain.ValidID(instance) || len(token) != 43 {
		return Record{}, storage.Record{}, nil, ErrDenied
	}
	row, err := s.Store.Get(ctx, Bucket, id)
	if err != nil {
		return Record{}, row, nil, ErrDenied
	}
	record, err := storage.Decode[Record](row)
	if err != nil {
		return record, row, nil, ErrUnavailable
	}
	if record.Reference.ID != id || record.Reference.Project != row.Project || record.Revoked || !time.Now().Before(record.Deadline) || record.Instance != "" && record.Instance != instance {
		return record, row, nil, ErrDenied
	}
	expected, err := s.token(ctx, record)
	if err != nil {
		return record, row, nil, err
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		return record, row, nil, ErrDenied
	}
	if s.Authorize == nil {
		return record, row, nil, ErrDenied
	}
	access, err := s.Authorize(ctx, record.Reference)
	if err != nil || !time.Now().Before(access.Deadline) {
		return record, row, nil, ErrDenied
	}
	return record, row, access.Records, nil
}

func (s *Service) save(ctx context.Context, row storage.Record, record Record, guards []storage.Record) error {
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, record)
	if err != nil {
		return ErrUnavailable
	}
	changes := []storage.Change{change}
	for _, guard := range guards {
		changes = append(changes, storage.Change{Record: guard, Assert: true})
	}
	if err := s.Store.Commit(ctx, changes...); errors.Is(err, storage.ErrConflict) {
		return ErrConflict
	} else if err != nil {
		return ErrUnavailable
	}
	return nil
}

func (t Ticket) String() string   { return "worker ticket: " + t.Execution }
func (t Ticket) GoString() string { return t.String() }
