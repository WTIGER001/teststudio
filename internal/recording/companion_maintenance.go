package recording

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) maintainCompanion(ctx context.Context, row storage.Record, value Session) {
	private, err := s.Store.Get(ctx, companionBucket, row.ID)
	if err != nil {
		return
	}
	connection, err := storage.Decode[companionRecord](private)
	if err != nil || private.Project != value.Project || connection.Binding != companionBinding(value) {
		return
	}
	stale := connection.Instance != "" && time.Since(connection.LastSeen) > CompanionLease
	if !stale && s.companionAuthority(ctx, value) == nil {
		return
	}
	value.State, value.Error = "failed", "The recording companion disconnected, expired, or lost current source permission. Captured code remains private to its owner."
	connection.Action = "cancel"
	publicChange, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, value)
	if err != nil {
		return
	}
	privateChange, err := storage.Put(private.Bucket, private.ID, private.Project, private.Revision, connection)
	if err != nil {
		return
	}
	_ = s.Store.Commit(ctx, publicChange, privateChange)
}
