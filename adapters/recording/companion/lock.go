package companion

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func (j *Journal) Lock() (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(j.directory, ".recorder.lock"), os.O_CREATE|os.O_RDWR|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, errors.New("private recorder lock is unavailable")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		file.Close()
		return nil, errors.New("private recorder lock is unsafe")
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		return nil, errors.New("another desktop companion is active in this recording directory")
	}
	return file, nil
}
