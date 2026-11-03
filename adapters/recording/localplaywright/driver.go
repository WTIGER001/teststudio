package localplaywright

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
)

type process struct {
	command  *exec.Cmd
	input    io.WriteCloser
	done     chan struct{}
	err      error
	stopping bool
}

type Driver struct {
	Node      string
	CLI       string
	Helper    string
	Directory string
	mu        sync.Mutex
	processes map[string]*process
}

func New(ctx context.Context, node, cli, helper, directory string) (*Driver, error) {
	var err error
	for _, value := range []*string{&cli, &helper, &directory} {
		*value, err = filepath.Abs(*value)
		if err != nil {
			return nil, err
		}
	}
	for _, file := range []string{cli, helper} {
		info, err := os.Lstat(file)
		if err != nil || !info.Mode().IsRegular() {
			return nil, errors.New("recording requires regular trusted Playwright CLI and helper files")
		}
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("recording directory must be a private regular directory")
	}
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(probe, node, cli, "--version").Output()
	if err != nil || strings.TrimSpace(string(output)) != "Version "+recording.PlaywrightVersion {
		return nil, errors.New("local recording requires the trusted runtime's exact Playwright CLI version")
	}
	return &Driver{Node: node, CLI: cli, Helper: helper, Directory: directory, processes: map[string]*process{}}, nil
}

func (d *Driver) Start(ctx context.Context, id, address string) error {
	if !domain.ValidID(id) || !recording.ValidURL(address) {
		return errors.New("invalid local browser recording request")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.processes[id]; exists {
		return errors.New("recording identifier is already used")
	}
	for _, item := range d.processes {
		select {
		case <-item.done:
		default:
			return recording.ErrBusy
		}
	}
	directory := filepath.Join(d.Directory, id)
	if err := os.Mkdir(directory, 0700); err != nil {
		return errors.New("recording output directory is already used or unavailable")
	}
	output := filepath.Join(directory, "source.spec.ts")
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(directory, "recorder.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	command := exec.Command(d.Node, d.Helper, d.CLI, output, address)
	command.Stdout, command.Stderr = log, log
	command.Env = []string{}
	for _, name := range []string{"PATH", "HOME", "TMPDIR", "DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "XDG_RUNTIME_DIR", "PLAYWRIGHT_BROWSERS_PATH", "SystemRoot", "LOCALAPPDATA"} {
		if value, ok := os.LookupEnv(name); ok {
			command.Env = append(command.Env, name+"="+value)
		}
	}
	input, err := command.StdinPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		input.Close()
		return errors.New("the local browser recorder could not start")
	}
	item := &process{command: command, input: input, done: make(chan struct{})}
	d.processes[id] = item
	go func() {
		err := command.Wait()
		d.mu.Lock()
		item.err = err
		close(item.done)
		d.mu.Unlock()
		input.Close()
	}()
	return nil
}

func (d *Driver) Read(ctx context.Context, id string) (recording.Snapshot, error) {
	if !domain.ValidID(id) {
		return recording.Snapshot{}, errors.New("invalid recorder identifier")
	}
	if err := ctx.Err(); err != nil {
		return recording.Snapshot{}, err
	}
	d.mu.Lock()
	item := d.processes[id]
	state, message := "ready", ""
	if item != nil {
		select {
		case <-item.done:
			if item.err != nil && !item.stopping {
				state, message = "failed", "The browser recorder stopped unexpectedly. Verify the local browser installation and review captured steps."
			}
		default:
			state = "recording"
		}
	} else {
		message = "The recorder service restarted. Review the source captured before it stopped."
	}
	d.mu.Unlock()
	filename := filepath.Join(d.Directory, id, "source.spec.ts")
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() > recording.ContentLimit {
		if err := d.closeProcess(ctx, id); err != nil {
			return recording.Snapshot{}, err
		}
		return recording.Snapshot{State: "failed", Error: "Recorded source is missing, unsafe, or exceeds its 256 KiB limit. The owned recorder stopped. Previously captured code remains retained."}, nil
	}
	data, err := os.ReadFile(filename)
	if err != nil || !utf8.Valid(data) {
		return recording.Snapshot{}, errors.New("recorded source cannot be read as UTF-8")
	}
	if len(data) == 0 && state == "ready" {
		state, message = "failed", "The browser closed before any Playwright source was recorded."
	}
	return recording.Snapshot{State: state, Code: string(data), Error: message}, nil
}

func (d *Driver) Stop(ctx context.Context, id string) (recording.Snapshot, error) {
	if err := d.closeProcess(ctx, id); err != nil {
		return recording.Snapshot{}, err
	}
	return d.Read(ctx, id)
}

func (d *Driver) closeProcess(ctx context.Context, id string) error {
	if !domain.ValidID(id) {
		return errors.New("invalid recorder identifier")
	}
	d.mu.Lock()
	item := d.processes[id]
	if item != nil {
		item.stopping = true
		item.input.Close()
	}
	d.mu.Unlock()
	if item != nil {
		select {
		case <-item.done:
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(12 * time.Second):
			return errors.New("the owned recorder has not stopped; cleanup remains pending")
		}
	}
	return nil
}

func (d *Driver) Close() error {
	d.mu.Lock()
	ids := []string{}
	for id := range d.processes {
		ids = append(ids, id)
	}
	d.mu.Unlock()
	for _, id := range ids {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_, err := d.Stop(ctx, id)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}
