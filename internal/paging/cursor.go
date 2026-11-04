package paging

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/wtiger001/teststudio/internal/storage"
	"io"
)

func Open(ctx context.Context, store *storage.Store, namespace, scope, cursor string) (string, error) {
	raw := ""
	if cursor != "" {
		if len(cursor) > 6000 {
			return "", errors.New("cursor does not match the query")
		}
		aead, err := Cipher(ctx, store, namespace)
		if err != nil {
			return "", err
		}
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(data) < aead.NonceSize() || len(data) > 4096 {
			return "", errors.New("cursor does not match the query")
		}
		decoded, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(scope))
		if err != nil {
			return "", errors.New("cursor does not match the query")
		}
		raw = string(decoded)
	}
	return raw, nil
}

func Seal(ctx context.Context, store *storage.Store, namespace, scope, raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	aead, err := Cipher(ctx, store, namespace)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(raw), []byte(scope))), nil
}

func Cipher(ctx context.Context, store *storage.Store, namespace string) (cipher.AEAD, error) {
	for range 3 {
		r, err := store.Get(ctx, namespace, "cursor-key")
		if errors.Is(err, storage.ErrNotFound) {
			key := make([]byte, 32)
			if _, err := io.ReadFull(rand.Reader, key); err != nil {
				return nil, err
			}
			c, err := storage.Put(namespace, "cursor-key", "", 0, key)
			if err != nil {
				return nil, err
			}
			if err := store.Commit(ctx, c); errors.Is(err, storage.ErrConflict) {
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
			return nil, errors.New("pagination cursor key is unavailable")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	}
	return nil, errors.New("pagination cursor key is unavailable")
}
