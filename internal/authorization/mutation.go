package authorization

import (
	"context"
	"github.com/wtiger001/teststudio/internal/storage"
)

type authorityRevisionKey struct{}

func (s *Service) BeginMutation(ctx context.Context) (context.Context, error) {
	version, err := s.Directory.Revision(ctx)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, authorityRevisionKey{}, version), nil
}

func AuthorityAssertions(ctx context.Context) []storage.Change {
	version, ok := ctx.Value(authorityRevisionKey{}).(storage.Record)
	if !ok || version.Revision == 0 {
		return nil
	}
	return []storage.Change{{Record: version, Assert: true}}
}
