package workerexchange

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

type quota struct {
	Bytes int64 `json:"bytes"`
}

func (s *Service) quotaChange(ctx context.Context, record Record, bucket, id string, amount, limit int64) (storage.Change, error) {
	row, err := s.Store.Get(ctx, bucket, id)
	value := quota{}
	if err == nil {
		if row.Project != record.Reference.Project {
			return storage.Change{}, ErrDenied
		}
		value, err = storage.Decode[quota](row)
		if err != nil {
			return storage.Change{}, ErrUnavailable
		}
	} else if errors.Is(err, storage.ErrNotFound) {
		row = storage.Record{Bucket: bucket, ID: id, Project: record.Reference.Project}
	} else {
		return storage.Change{}, ErrUnavailable
	}
	if value.Bytes < 0 || amount < 0 && value.Bytes < -amount {
		return storage.Change{}, ErrUnavailable
	}
	value.Bytes += amount
	if value.Bytes < 0 || amount > 0 && (limit < 1 || value.Bytes > limit) {
		return storage.Change{}, ErrQuota
	}
	if value.Bytes == 0 && row.Revision > 0 {
		return storage.Change{Record: row, Delete: true}, nil
	}
	change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err != nil {
		return storage.Change{}, ErrUnavailable
	}
	return change, nil
}

func (s *Service) reserve(ctx context.Context, record Record, access workerstate.Access) ([]storage.Change, error) {
	if record.StagingBytes == 0 {
		return nil, nil
	}
	project, err := s.quotaChange(ctx, record, "worker-exchange-project-quota", record.Reference.Project, record.StagingBytes, access.ProjectBytes)
	if err != nil {
		return nil, err
	}
	run, err := s.quotaChange(ctx, record, "worker-exchange-run-quota", record.Reference.Run, record.StagingBytes, access.RunBytes)
	if err != nil {
		return nil, err
	}
	return []storage.Change{project, run}, nil
}

func (s *Service) release(ctx context.Context, record Record) ([]storage.Change, error) {
	if record.StagingBytes == 0 {
		return nil, nil
	}
	project, err := s.quotaChange(ctx, record, "worker-exchange-project-quota", record.Reference.Project, -record.StagingBytes, 0)
	if err != nil {
		return nil, err
	}
	run, err := s.quotaChange(ctx, record, "worker-exchange-run-quota", record.Reference.Run, -record.StagingBytes, 0)
	if err != nil {
		return nil, err
	}
	return []storage.Change{project, run}, nil
}
