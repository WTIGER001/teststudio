package registryimage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
)

type uploadState struct {
	Owner   execution.Reference `json:"owner"`
	Runtime string              `json:"runtime"`
	Opening string              `json:"opening,omitempty"`
	Upload  string              `json:"upload,omitempty"`
}

type uploadJournal struct {
	root  *os.Root
	state uploadState
}

func journalName(ref execution.Reference) string {
	data, _ := json.Marshal(ref)
	hash := sha256.Sum256(data)
	return "registry-source-" + hex.EncodeToString(hash[:])
}

func privateDirectory(info os.FileInfo) bool {
	return info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0077 == 0
}

func (b *Builder) openDirectory(create bool) (*os.Root, error) {
	if b.Directory == "" {
		return nil, errors.New("registry assembly needs a private recovery directory")
	}
	directory, err := filepath.Abs(b.Directory)
	if err != nil {
		return nil, errors.New("registry assembly directory is invalid")
	}
	if create {
		if err := os.MkdirAll(directory, 0700); err != nil {
			return nil, errors.New("registry assembly directory cannot be created")
		}
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !privateDirectory(info) {
		return nil, errors.New("registry assembly directory must be private and cannot be a symbolic link")
	}
	return os.OpenRoot(directory)
}

func (b *Builder) reserve(input preparation.BuildRequest) (*uploadJournal, *os.Root, error) {
	parent, err := b.openDirectory(true)
	if err != nil {
		return nil, nil, err
	}
	name := journalName(input.Reference)
	if err := parent.Mkdir(name, 0700); err != nil {
		parent.Close()
		return nil, nil, errors.New("registry assembly reference already exists or needs recovery")
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		parent.Close()
		return nil, nil, errors.New("registry assembly journal cannot be opened")
	}
	journal := &uploadJournal{root: root, state: uploadState{Owner: input.Reference, Runtime: b.RuntimeImage}}
	if err := errors.Join(journal.save(), syncDirectory(parent)); err != nil {
		root.Close()
		parent.RemoveAll(name)
		parent.Close()
		return nil, nil, err
	}
	return journal, parent, nil
}

func readJournal(parent *os.Root, ref execution.Reference) (*uploadJournal, error) {
	name := journalName(ref)
	info, err := parent.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !privateDirectory(info) {
		return nil, errors.New("registry cleanup refuses an unowned recovery directory")
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	file, err := root.OpenFile("state.json", os.O_RDONLY, 0)
	if err != nil {
		root.Close()
		return nil, errors.New("registry cleanup journal is unavailable")
	}
	defer file.Close()
	info, err = file.Stat()
	data, readErr := io.ReadAll(io.LimitReader(file, 16385))
	journal := &uploadJournal{root: root}
	if err != nil || readErr != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || len(data) > 16384 || json.Unmarshal(data, &journal.state) != nil || journal.state.Owner != ref {
		root.Close()
		return nil, errors.New("registry cleanup journal does not prove private ownership")
	}
	if journal.state.Opening != "" && journal.state.Upload == "" {
		pending, err := root.Open("state.pending")
		if err == nil {
			defer pending.Close()
			info, statErr := pending.Stat()
			data, readErr := io.ReadAll(io.LimitReader(pending, 16385))
			var next uploadState
			if statErr == nil && readErr == nil && info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0 && len(data) <= 16384 && json.Unmarshal(data, &next) == nil && next.Owner == ref && next.Runtime == journal.state.Runtime && next.Opening == journal.state.Opening && next.Upload != "" {
				journal.state.Upload = next.Upload
			}
		}
	}
	return journal, nil
}

func (j *uploadJournal) save() error {
	data, err := json.Marshal(j.state)
	if err != nil || len(data) > 16384 {
		return errors.New("registry assembly journal exceeds its limit")
	}
	file, err := j.root.OpenFile("state.pending", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("registry assembly journal cannot be reserved")
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || j.root.Rename("state.pending", "state.json") != nil {
		return errors.New("registry assembly journal could not be committed")
	}
	return syncDirectory(j.root)
}

func syncDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return errors.New("registry assembly journal directory is unavailable")
	}
	defer directory.Close()
	if directory.Sync() != nil {
		return errors.New("registry assembly journal directory could not be committed")
	}
	return nil
}

func (j *uploadJournal) Opening(hash string) error {
	if j.state.Opening != "" || j.state.Upload != "" {
		return preparation.ErrCleanupIncomplete
	}
	j.state.Opening = hash
	return j.save()
}

func (j *uploadJournal) Opened(location string) error {
	if j.state.Opening == "" || j.state.Upload != "" {
		return preparation.ErrCleanupIncomplete
	}
	j.state.Upload = location
	return j.save()
}

func (j *uploadJournal) Completed() error {
	j.state.Opening, j.state.Upload = "", ""
	return j.save()
}
