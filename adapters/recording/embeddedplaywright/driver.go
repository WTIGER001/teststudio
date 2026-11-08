package embeddedplaywright

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/adapters/recording/localplaywright"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
)

type response struct {
	View  recording.View `json:"view"`
	Code  string         `json:"code"`
	Error string         `json:"error"`
}

type process struct {
	mu        sync.Mutex
	command   *exec.Cmd
	input     io.WriteCloser
	responses chan response
	done      chan struct{}
	snapshot  recording.Snapshot
}

type Driver struct {
	runtime *localplaywright.Driver
	mu      sync.Mutex
	items   map[string]*process
}

func New(ctx context.Context, node, cli, helper, directory string) (*Driver, error) {
	runtime, err := localplaywright.New(ctx, node, cli, helper, directory)
	if err != nil {
		return nil, err
	}
	return &Driver{runtime: runtime, items: map[string]*process{}}, nil
}

func (d *Driver) Start(ctx context.Context, id, address string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !domain.ValidID(id) || !recording.ValidURL(address) {
		return errors.New("invalid embedded browser recording request")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.items[id] != nil {
		return errors.New("recording identifier is already used")
	}
	for _, item := range d.items {
		select {
		case <-item.done:
		default:
			return recording.ErrBusy
		}
	}
	directory := filepath.Join(d.runtime.Directory, id)
	if err := os.Mkdir(directory, 0700); err != nil {
		return errors.New("recording output directory is already used or unavailable")
	}
	log, err := os.OpenFile(filepath.Join(directory, "recorder.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	command := exec.Command(d.runtime.Node, d.runtime.Helper, d.runtime.CLI, filepath.Join(directory, "source.spec.ts"), address)
	command.Stderr = log
	command.Env = []string{}
	for _, name := range []string{"PATH", "HOME", "TMPDIR", "PLAYWRIGHT_BROWSERS_PATH", "SystemRoot", "LOCALAPPDATA"} {
		if value, ok := os.LookupEnv(name); ok {
			command.Env = append(command.Env, name+"="+value)
		}
	}
	input, err := command.StdinPipe()
	if err != nil {
		return err
	}
	output, err := command.StdoutPipe()
	if err != nil {
		input.Close()
		return err
	}
	if err := command.Start(); err != nil {
		input.Close()
		output.Close()
		return errors.New("the recording browser could not start")
	}
	item := &process{command: command, input: input, responses: make(chan response, 1), done: make(chan struct{}), snapshot: recording.Snapshot{State: "recording"}}
	d.items[id] = item
	go func() {
		defer close(item.done)
		lines := bufio.NewScanner(output)
		lines.Buffer(make([]byte, 64<<10), 4<<20)
		for lines.Scan() {
			var value response
			if json.Unmarshal(lines.Bytes(), &value) != nil {
				break
			}
			select {
			case item.responses <- value:
			default:
				input.Close()
			}
		}
		input.Close()
		_ = command.Wait()
	}()
	_, err = item.exchange(ctx, recording.Action{Action: "view"})
	if err != nil {
		input.Close()
	}
	return err
}

func (d *Driver) item(id string) (*process, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !domain.ValidID(id) || d.items[id] == nil {
		return nil, errors.New("the recording browser is unavailable; start a new recording")
	}
	return d.items[id], nil
}

func (p *process) exchange(ctx context.Context, input recording.Action) (recording.View, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	select {
	case <-p.done:
		return recording.View{}, errors.New("the recording browser closed; review its retained steps")
	default:
	}
	if err := json.NewEncoder(p.input).Encode(input); err != nil {
		return recording.View{}, errors.New("the recording browser connection closed")
	}
	select {
	case value := <-p.responses:
		if len(value.Code) > recording.ContentLimit || len(value.View.Steps) > 500 {
			p.input.Close()
			return recording.View{}, errors.New("the recording reached its source or step limit")
		}
		p.snapshot.Code = value.Code
		if value.Error != "" {
			return recording.View{}, errors.New(value.Error)
		}
		p.snapshot.Steps = value.View.Steps
		return value.View, nil
	case <-p.done:
		return recording.View{}, errors.New("the recording browser closed; review its retained steps")
	case <-ctx.Done():
		p.input.Close()
		return recording.View{}, ctx.Err()
	}
}

func (d *Driver) View(ctx context.Context, id string) (recording.View, error) {
	item, err := d.item(id)
	if err != nil {
		return recording.View{}, err
	}
	return item.exchange(ctx, recording.Action{Action: "view"})
}

func (d *Driver) Interact(ctx context.Context, id string, input recording.Action) (recording.View, error) {
	if !input.Valid() {
		return recording.View{}, errors.New("unsupported browser recording action")
	}
	item, err := d.item(id)
	if err != nil {
		return recording.View{}, err
	}
	return item.exchange(ctx, input)
}

func (d *Driver) Read(ctx context.Context, id string) (recording.Snapshot, error) {
	item, err := d.item(id)
	if err != nil {
		return recording.Snapshot{State: "failed", Error: "The recording service restarted. Previously captured source remains private in recording history."}, nil
	}
	item.mu.Lock()
	defer item.mu.Unlock()
	value := item.snapshot
	select {
	case <-item.done:
		value.State = "failed"
		value.Error = "The recording browser closed. Review its captured steps before starting a new recording."
	default:
	}
	return value, ctx.Err()
}

func (d *Driver) Stop(ctx context.Context, id string) (recording.Snapshot, error) {
	item, err := d.item(id)
	if err != nil {
		return d.Read(ctx, id)
	}
	_, _ = item.exchange(ctx, recording.Action{Action: "view"})
	item.mu.Lock()
	item.input.Close()
	item.mu.Unlock()
	select {
	case <-item.done:
	case <-ctx.Done():
		return recording.Snapshot{}, ctx.Err()
	case <-time.After(12 * time.Second):
		return recording.Snapshot{}, errors.New("the recording browser is still closing; cleanup remains pending")
	}
	item.mu.Lock()
	defer item.mu.Unlock()
	value := item.snapshot
	value.State, value.Error = "ready", ""
	if value.Code == "" {
		value.State, value.Error = "failed", "The recording ended before any steps were captured."
	}
	return value, nil
}

func (d *Driver) Close() error {
	d.mu.Lock()
	ids := make([]string, 0, len(d.items))
	for id := range d.items {
		ids = append(ids, id)
	}
	d.mu.Unlock()
	var result error
	for _, id := range ids {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_, err := d.Stop(ctx, id)
		cancel()
		result = errors.Join(result, err)
	}
	return result
}
