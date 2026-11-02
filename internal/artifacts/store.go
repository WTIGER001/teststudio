package artifacts

import (
	"context"
	"errors"
	"io"
)

var ErrNotFound = errors.New("artifact content is missing")

type Object struct {
	Size     int64
	Checksum string
}

type Store interface {
	Put(context.Context, string, io.Reader, int64) (Object, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
