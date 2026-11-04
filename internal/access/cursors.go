package access

import (
	"context"
	"crypto/cipher"
	"github.com/wtiger001/teststudio/internal/paging"
	"github.com/wtiger001/teststudio/internal/storage"
	"strconv"
)

func queryScope(actor, kind, resource string, archived bool) string {
	return actor + "\x00" + kind + "\x00" + resource + "\x00" + strconv.FormatBool(archived)
}

func (s *Service) page(ctx context.Context, bucket, project, scope, cursor string, limit int) (storage.Page, error) {
	raw, err := s.openCursor(ctx, scope, cursor)
	if err != nil {
		return storage.Page{}, err
	}
	page, err := s.Store.List(ctx, bucket, project, raw, limit)
	if err != nil || page.Cursor == "" {
		return page, err
	}
	page.Cursor, err = s.sealCursor(ctx, scope, page.Cursor)
	return page, err
}

func (s *Service) openCursor(ctx context.Context, scope, cursor string) (string, error) {
	return paging.Open(ctx, s.Store, "access-settings", scope, cursor)
}
func (s *Service) sealCursor(ctx context.Context, scope, raw string) (string, error) {
	return paging.Seal(ctx, s.Store, "access-settings", scope, raw)
}

func (s *Service) cursorCipher(ctx context.Context) (cipher.AEAD, error) {
	return paging.Cipher(ctx, s.Store, "access-settings")
}
