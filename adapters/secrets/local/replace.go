package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/wtiger001/teststudio/internal/secrets"
)

func (p *Provider) Replaceable(project, id, purpose string) bool {
	b, ok := p.bindings[project+"/"+id]
	if !ok || b.Purpose != purpose || b.File == "" {
		return false
	}
	info, err := os.Lstat(b.File)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0200 == 0 {
		return false
	}
	return syscall.Access(filepath.Dir(b.File), 2) == nil
}
func (p *Provider) Replace(ctx context.Context, project, id, purpose string, expected, replacement []byte) error {
	if !p.Replaceable(project, id, purpose) || len(replacement) < 1 || len(replacement) > 65536 {
		return secrets.ErrUnavailable
	}
	b := p.bindings[project+"/"+id]
	lock, err := os.OpenFile(b.File+".rotation-lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return secrets.ErrUnavailable
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return errors.New("credential replacement is already in progress")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if err = ctx.Err(); err != nil {
		return err
	}
	info, err := os.Lstat(b.File)
	if err != nil || !info.Mode().IsRegular() {
		return secrets.ErrUnavailable
	}
	current, err := p.Resolve(ctx, project, id, purpose)
	if err != nil || !bytes.Equal(current.Bytes, expected) {
		return errors.New("credential changed during replacement; inspect the binding")
	}
	file, err := os.CreateTemp(filepath.Dir(b.File), ".teststudio-secret-*")
	if err != nil {
		return secrets.ErrUnavailable
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err = file.Chmod(0600); err != nil {
		return secrets.ErrUnavailable
	}
	if _, err = file.Write(replacement); err != nil {
		return secrets.ErrUnavailable
	}
	if err = file.Sync(); err != nil {
		return secrets.ErrUnavailable
	}
	if err = file.Close(); err != nil {
		return secrets.ErrUnavailable
	}
	if err = os.Rename(file.Name(), b.File); err != nil {
		return secrets.ErrUnavailable
	}
	directory, err := os.Open(filepath.Dir(b.File))
	if err != nil {
		return secrets.ErrUnavailable
	}
	defer directory.Close()
	if err = directory.Sync(); err != nil {
		return secrets.ErrUnavailable
	}
	return nil
}
