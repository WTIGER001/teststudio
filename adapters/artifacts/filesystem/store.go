package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
)

type Object struct {
	Size     int64
	Checksum string
}

type Store struct{ root *os.Root }

func New(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func (s *Store) Close() error { return s.root.Close() }

func validKey(key string) bool {
	return key != "" && !strings.HasPrefix(key, "/") && !strings.Contains(key, "\\") && path.Clean(key) == key && key != ".." && !strings.HasPrefix(key, "../")
}

func (s *Store) Put(ctx context.Context, key string, source io.Reader, limit int64) (Object, error) {
	if !validKey(key) || limit < 1 || limit > 1<<30 {
		return Object{}, errors.New("invalid artifact key or size limit")
	}
	if err := ctx.Err(); err != nil {
		return Object{}, err
	}
	if err := s.root.MkdirAll(path.Dir(key), 0700); err != nil {
		return Object{}, err
	}
	tmp := key + ".upload-" + domain.NewID()
	f, err := s.root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Object{}, err
	}
	defer s.root.Remove(tmp)
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(f, hash), io.LimitReader(source, limit+1))
	if copyErr == nil {
		copyErr = ctx.Err()
	}
	if copyErr == nil && size > limit {
		copyErr = errors.New("artifact exceeds its byte limit")
	}
	if copyErr == nil {
		copyErr = f.Sync()
	}
	closeErr := f.Close()
	if copyErr != nil {
		return Object{}, copyErr
	}
	if closeErr != nil {
		return Object{}, closeErr
	}
	if err := s.root.Rename(tmp, key); err != nil {
		return Object{}, err
	}
	return Object{Size: size, Checksum: hex.EncodeToString(hash.Sum(nil))}, nil
}

func (s *Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !validKey(key) {
		return nil, errors.New("invalid artifact key")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.root.Open(key)
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if !validKey(key) {
		return errors.New("invalid artifact key")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err := s.root.Remove(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
