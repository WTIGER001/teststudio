package timing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct{ Store *storage.Store }

func Key(v Context) string {
	data, _ := json.Marshal(v)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *Service) Prediction(ctx context.Context, v Context, p Policy, hardTimeoutMs int, asOf time.Time) (Prediction, error) {
	if err := Validate(p); err != nil {
		return Prediction{}, err
	}
	w := Window{Context: v, Generation: 1, Samples: []Sample{}}
	r, err := s.Store.Get(ctx, "timing-windows", Key(v))
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return Prediction{}, err
	}
	if err == nil {
		w, err = storage.Decode[Window](r)
		if err != nil {
			return Prediction{}, err
		}
	}
	return Predict(w, r.Revision, p, hardTimeoutMs, asOf), nil
}

func (s *Service) RecordChanges(ctx context.Context, p Prediction, sample Sample) ([]storage.Change, error) {
	if !finite(sample.DurationMs) || sample.DurationMs < 0 || sample.Run == "" || sample.At.IsZero() {
		return nil, nil
	}
	r, err := s.Store.Get(ctx, "timing-windows", p.Key)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	w := Window{Context: p.Context, Generation: 1, Samples: []Sample{}}
	if err == nil {
		w, err = storage.Decode[Window](r)
		if err != nil {
			return nil, err
		}
	}
	if w.Generation != p.Generation {
		return nil, nil
	}
	for _, old := range w.Samples {
		if old.Run == sample.Run {
			return nil, nil
		}
	}
	w.Samples = append(w.Samples, sample)
	slices.SortFunc(w.Samples, func(a, b Sample) int {
		if n := a.At.Compare(b.At); n != 0 {
			return n
		}
		if a.Run < b.Run {
			return -1
		}
		if a.Run > b.Run {
			return 1
		}
		return 0
	})
	if len(w.Samples) > 1000 {
		w.Samples = w.Samples[len(w.Samples)-1000:]
	}
	change, err := storage.Put("timing-windows", p.Key, p.Context.Project, r.Revision, w)
	return []storage.Change{change}, err
}

func (s *Service) Reset(ctx context.Context, user, project, id, reason string, revision int64) error {
	r, err := s.Store.Get(ctx, "timing-windows", id)
	if err != nil {
		return err
	}
	w, err := storage.Decode[Window](r)
	if err != nil {
		return err
	}
	if w.Context.Project != project {
		return storage.ErrNotFound
	}
	if r.Revision != revision {
		return storage.ErrConflict
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 500 {
		return errors.New("timing baseline reset requires a reason of up to 500 characters")
	}
	before := w
	now := time.Now().UTC()
	w.Generation++
	w.Samples = []Sample{}
	w.ResetAt = &now
	w.ResetReason = reason
	change, err := storage.Put(r.Bucket, r.ID, project, r.Revision, w)
	if err != nil {
		return err
	}
	audit, err := domain.Audit(user, project, "timing.reset", id, before, w)
	if err != nil {
		return err
	}
	return s.Store.Commit(ctx, change, audit)
}
