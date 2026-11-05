package companion

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/recording"
)

type remoteFixture struct {
	packet       recording.CompanionPacket
	pair         recording.PairRequest
	control      recording.CompanionControl
	controls     int
	submitted    []recording.CompanionSnapshot
	losePair     bool
	loseSnapshot bool
	revoke       bool
}

func (f *remoteFixture) Pair(ctx context.Context, id string, input recording.PairRequest) (recording.CompanionPacket, error) {
	if f.pair.Instance != "" && f.pair != input {
		return recording.CompanionPacket{}, ErrAuthority
	}
	f.pair = input
	if f.losePair {
		f.losePair = false
		return recording.CompanionPacket{}, ErrTransport
	}
	return f.packet, ctx.Err()
}
func (f *remoteFixture) Control(ctx context.Context, _, instance, token string) (recording.CompanionControl, error) {
	if instance != f.pair.Instance || token != f.pair.DeviceToken {
		return recording.CompanionControl{}, ErrAuthority
	}
	f.controls++
	if f.controls > 1 {
		if f.revoke {
			return recording.CompanionControl{}, ErrAuthority
		}
		f.control.Action = "stop"
	}
	return f.control, ctx.Err()
}
func (f *remoteFixture) Submit(ctx context.Context, _, instance, token string, input recording.CompanionSnapshot) (recording.CompanionControl, error) {
	if instance != f.pair.Instance || token != f.pair.DeviceToken {
		return recording.CompanionControl{}, ErrAuthority
	}
	if input.Sequence == f.control.Sequence {
		if len(f.submitted) == 0 || input != f.submitted[len(f.submitted)-1] {
			return recording.CompanionControl{}, errors.New("changed snapshot retry")
		}
		return f.control, ctx.Err()
	}
	if input.Sequence != f.control.Sequence+1 {
		return recording.CompanionControl{}, errors.New("out of order snapshot")
	}
	f.submitted = append(f.submitted, input)
	f.control.Sequence, f.control.State = input.Sequence, input.State
	if f.loseSnapshot {
		f.loseSnapshot = false
		return recording.CompanionControl{}, ErrTransport
	}
	return f.control, ctx.Err()
}

type driverFixture struct {
	starts, stops int
	source        string
}

func (d *driverFixture) Start(context.Context, string, string) error { d.starts++; return nil }
func (d *driverFixture) Read(context.Context, string) (recording.Snapshot, error) {
	return recording.Snapshot{State: "recording", Code: d.source}, nil
}
func (d *driverFixture) Stop(context.Context, string) (recording.Snapshot, error) {
	d.stops++
	return recording.Snapshot{State: "ready", Code: "final captured steps"}, nil
}

func newRunnerFixture(t *testing.T) (*Runner, *State, *remoteFixture, *driverFixture) {
	t.Helper()
	token, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	state, err := NewState("https://studio.example.test", "session:"+token)
	if err != nil {
		t.Fatal(err)
	}
	j, err := NewJournal(filepath.Join(t.TempDir(), "connections"))
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Create(state); err != nil {
		t.Fatal(err)
	}
	control := recording.CompanionControl{State: "starting", DeadlineAt: time.Now().Add(time.Minute)}
	remote := &remoteFixture{control: control, packet: recording.CompanionPacket{Input: recording.Input{ID: "session", URL: "https://example.test"}, Project: "example", Suite: "browser", Control: control}}
	driver := &driverFixture{source: "initial captured steps"}
	return &Runner{Remote: remote, Driver: driver, Journal: j, Interval: time.Millisecond}, &state, remote, driver
}

func TestDesktopCompanionRecoversPairAcknowledgementAndStopsOnOwnerRequest(t *testing.T) {
	r, state, remote, driver := newRunnerFixture(t)
	remote.losePair = true
	if err := r.Run(t.Context(), state); !errors.Is(err, ErrTransport) || driver.starts != 0 {
		t.Fatal("uncertain pairing launched a recorder", err)
	}
	saved, err := r.Journal.Read(state.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(t.Context(), &saved); err != nil {
		t.Fatal(err)
	}
	if driver.starts != 1 || driver.stops != 1 || remote.control.State != "ready" || len(remote.submitted) != 2 || remote.submitted[1].Code != "final captured steps" {
		t.Fatal("record and stop workflow lost owned source")
	}
	saved, err = r.Journal.Read(state.ID)
	if err != nil || saved.PairingToken != "" || saved.Sequence != 2 || saved.Final == nil {
		t.Fatal("durable final connection is incomplete", err)
	}
}

func TestDesktopCompanionLostSnapshotStopsBrowserAndRetriesWithoutRelaunch(t *testing.T) {
	r, state, remote, driver := newRunnerFixture(t)
	remote.loseSnapshot = true
	if err := r.Run(t.Context(), state); !errors.Is(err, ErrTransport) {
		t.Fatal(err)
	}
	if driver.starts != 1 || driver.stops != 1 {
		t.Fatal("transport loss left an owned recorder running")
	}
	saved, err := r.Journal.Read(state.ID)
	if err != nil || saved.Pending == nil || saved.Pending.Sequence != 1 || saved.Final == nil || saved.Final.Code != "final captured steps" {
		t.Fatal("lost acknowledgement discarded source or retry", err)
	}
	recoveredDriver := &driverFixture{}
	r.Driver = recoveredDriver
	if err := r.Run(t.Context(), &saved); err != nil {
		t.Fatal(err)
	}
	if recoveredDriver.starts != 0 || len(remote.submitted) != 2 || remote.submitted[1].Sequence != 2 || remote.submitted[1].Failure != "transport" {
		t.Fatal("recovery replayed launch or duplicated captured steps")
	}
}

func TestDesktopCompanionRecoversAttemptedLaunchWithoutOpeningBrowser(t *testing.T) {
	r, state, remote, driver := newRunnerFixture(t)
	state.Packet, state.PairingToken, state.StartAttempted = &remote.packet, "", true
	remote.pair = recording.PairRequest{Instance: state.Instance, DeviceToken: state.DeviceToken}
	if err := r.Journal.Save(*state); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(t.Context(), state); err != nil {
		t.Fatal(err)
	}
	if driver.starts != 0 || driver.stops != 1 || remote.control.State != "ready" {
		t.Fatal("restart launched another recorder instead of recovering files")
	}
}

func TestDesktopCompanionRecoversFinalAcknowledgementWithoutReplacingTerminalSource(t *testing.T) {
	r, state, remote, driver := newRunnerFixture(t)
	state.Packet, state.PairingToken, state.StartAttempted = &remote.packet, "", true
	remote.pair = recording.PairRequest{Instance: state.Instance, DeviceToken: state.DeviceToken}
	final := recording.CompanionSnapshot{Sequence: 1, State: "ready", Code: "final captured steps"}
	state.Final, state.Pending = &final, &final
	remote.control.State, remote.control.Sequence = "ready", 1
	remote.submitted = []recording.CompanionSnapshot{final}
	if err := r.Journal.Save(*state); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(t.Context(), state); err != nil {
		t.Fatal(err)
	}
	if state.Pending != nil || state.Sequence != 1 || len(remote.submitted) != 1 || driver.starts != 0 {
		t.Fatal("final retry lost acknowledgement or replaced terminal source")
	}
}

func TestDesktopCompanionRevocationAndDeadlineStopOwnedRecorder(t *testing.T) {
	t.Run("revocation", func(t *testing.T) {
		r, state, remote, driver := newRunnerFixture(t)
		remote.revoke = true
		if err := r.Run(t.Context(), state); !errors.Is(err, ErrAuthority) {
			t.Fatal(err)
		}
		if driver.stops != 1 || state.Final == nil || len(remote.submitted) != 1 {
			t.Fatal("revoked authority retained browser or uploaded more source")
		}
	})
	t.Run("deadline", func(t *testing.T) {
		r, state, remote, driver := newRunnerFixture(t)
		remote.packet.Control.DeadlineAt, remote.control.DeadlineAt = time.Now().Add(10*time.Millisecond), time.Now().Add(10*time.Millisecond)
		remote.control.DeadlineAt = remote.packet.Control.DeadlineAt
		r.Interval = time.Second
		if err := r.Run(t.Context(), state); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
		if driver.starts != 1 || driver.stops != 1 || state.Final == nil {
			t.Fatal("deadline did not stop the owned recorder")
		}
	})
}
