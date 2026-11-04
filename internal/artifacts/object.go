package artifacts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"strings"
)

func ValidKey(key string) bool {
	return key != "" && key != "." && len(key) <= 1024 && !strings.HasPrefix(key, "/") && !strings.ContainsAny(key, "\\\x00\r\n") && path.Clean(key) == key && key != ".." && !strings.HasPrefix(key, "../")
}

func ValidPrefix(prefix string) bool { return prefix == "" || ValidKey(prefix) }

func ObjectKey(prefix, key string) (string, error) {
	if !ValidPrefix(prefix) || !ValidKey(key) {
		return "", errors.New("invalid artifact prefix or key")
	}
	if prefix != "" {
		key = prefix + "/" + key
	}
	if !ValidKey(key) {
		return "", errors.New("artifact key exceeds the storage limit")
	}
	return key, nil
}

type Staged struct {
	*os.File
	Object Object
}

func (s *Staged) Close() error {
	return errors.Join(s.File.Close(), os.Remove(s.File.Name()))
}

type contextReader struct {
	ctx    context.Context
	source io.Reader
}

func (r contextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.source.Read(data)
	if r.ctx.Err() != nil {
		return n, r.ctx.Err()
	}
	return n, err
}

func Stage(ctx context.Context, source io.Reader, limit int64) (*Staged, error) {
	if limit < 1 || limit > 1<<30 {
		return nil, errors.New("invalid artifact byte limit")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp("", "teststudio-artifact-*")
	if err != nil {
		return nil, err
	}
	staged := &Staged{File: file}
	complete := false
	defer func() {
		if !complete {
			_ = staged.Close()
		}
	}()
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(contextReader{ctx, source}, limit+1))
	if err != nil {
		return nil, err
	}
	if size > limit {
		return nil, errors.New("artifact exceeds its byte limit")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	staged.Object = Object{Size: size, Checksum: hex.EncodeToString(hash.Sum(nil))}
	complete = true
	return staged, nil
}
