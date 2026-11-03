package access

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strconv"

	"github.com/wtiger001/teststudio/internal/storage"
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
	raw := ""
	if cursor != "" {
		if len(cursor) > 6000 {
			return "", errors.New("cursor does not match the access query")
		}
		aead, err := s.cursorCipher(ctx)
		if err != nil {
			return "", err
		}
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(data) < aead.NonceSize() || len(data) > 4096 {
			return "", errors.New("cursor does not match the access query")
		}
		decoded, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(scope))
		if err != nil {
			return "", errors.New("cursor does not match the access query")
		}
		raw = string(decoded)
	}
	return raw, nil
}

func (s *Service) sealCursor(ctx context.Context, scope, raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	aead, err := s.cursorCipher(ctx)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(raw), []byte(scope))), nil
}

func (s *Service) cursorCipher(ctx context.Context) (cipher.AEAD, error) {
	for range 3 {
		r, err := s.Store.Get(ctx, "access-settings", "cursor-key")
		if errors.Is(err, storage.ErrNotFound) {
			key := make([]byte, 32)
			if _, err := io.ReadFull(rand.Reader, key); err != nil {
				return nil, err
			}
			c, err := storage.Put("access-settings", "cursor-key", "", 0, key)
			if err != nil {
				return nil, err
			}
			if err := s.Store.Commit(ctx, c); errors.Is(err, storage.ErrConflict) {
				continue
			} else if err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		key, err := storage.Decode[[]byte](r)
		if err != nil || len(key) != 32 {
			return nil, errors.New("access cursor key is unavailable")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	}
	return nil, errors.New("access cursor key is unavailable")
}
