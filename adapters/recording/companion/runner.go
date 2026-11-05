package companion

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/recording"
)

type Runner struct {
	Remote        Remote
	Driver        recording.Driver
	Journal       *Journal
	Interval      time.Duration
	RecoveryDelay time.Duration
}

func terminal(state string) bool {
	return state == "ready" || state == "failed" || state == "cancelled"
}

func (r *Runner) Run(ctx context.Context, state *State) (err error) {
	if r.Remote == nil || r.Driver == nil || r.Journal == nil {
		return errors.New("desktop recorder is not configured")
	}
	if state.Packet == nil {
		packet, pairErr := r.Remote.Pair(ctx, state.ID, recording.PairRequest{PairingToken: state.PairingToken, Instance: state.Instance, DeviceToken: state.DeviceToken})
		if pairErr != nil {
			return pairErr
		}
		if packet.Input.ID != state.ID || !recording.ValidURL(packet.Input.URL) || !domain.ValidID(packet.Project) || !domain.ValidID(packet.Suite) || packet.Control.DeadlineAt.After(time.Now().Add(recording.Lifetime+time.Minute)) {
			return ErrAuthority
		}
		state.Packet, state.PairingToken = &packet, ""
		if err := r.Journal.Save(*state); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithDeadline(ctx, state.Packet.Control.DeadlineAt)
	defer cancel()
	wasStarted := state.StartAttempted
	defer func() {
		if err != nil && state.StartAttempted && state.Final == nil {
			_ = r.captureFinal(state, "transport")
		}
	}()
	control, err := r.control(ctx, state)
	if err != nil {
		return err
	}
	if terminal(control.State) {
		if state.Pending != nil && control.Sequence == state.Pending.Sequence {
			if _, err := r.flush(ctx, state); err != nil {
				return err
			}
		}
		if wasStarted && state.Final == nil {
			return r.captureFinal(state, "")
		}
		return nil
	}
	if state.Pending != nil {
		control, err = r.flush(ctx, state)
		if err != nil {
			return err
		}
		if terminal(control.State) {
			return nil
		}
	}
	if wasStarted && state.Final == nil {
		if r.RecoveryDelay > 0 {
			timer := time.NewTimer(r.RecoveryDelay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
			if control, err = r.control(ctx, state); err != nil {
				return err
			}
			if terminal(control.State) {
				return r.captureFinal(state, "")
			}
		}
		if err := r.captureFinal(state, ""); err != nil {
			return err
		}
	}
	if state.Final != nil {
		return r.sendFinal(ctx, state)
	}
	if control.Action != "" {
		state.Final = &recording.CompanionSnapshot{State: "ready"}
		if err := r.Journal.Save(*state); err != nil {
			return err
		}
		return r.sendFinal(ctx, state)
	}
	state.StartAttempted = true
	if err := r.Journal.Save(*state); err != nil {
		return err
	}
	if err := r.Driver.Start(ctx, state.ID, state.Packet.Input.URL); err != nil {
		if err := r.captureFinal(state, "recorder"); err != nil {
			return err
		}
		return r.sendFinal(ctx, state)
	}
	interval := r.Interval
	if interval <= 0 || interval > 5*time.Second {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if control.Action != "" {
			if err := r.captureFinal(state, ""); err != nil {
				return err
			}
			return r.sendFinal(ctx, state)
		}
		snapshot, readErr := r.Driver.Read(ctx, state.ID)
		if readErr != nil {
			if err := r.captureFinal(state, "capture"); err != nil {
				return err
			}
			return r.sendFinal(ctx, state)
		}
		if snapshot.State != "recording" {
			if err := r.captureFinal(state, ""); err != nil {
				return err
			}
			return r.sendFinal(ctx, state)
		}
		update := recordedSnapshot(snapshot, "")
		if update.State == "recording" && update.Code == "" {
			update.Code = state.Last.Code
		}
		if update.State != "recording" {
			if err := r.captureFinal(state, "capture"); err != nil {
				return err
			}
			return r.sendFinal(ctx, state)
		}
		if update.State != state.Last.State || update.Code != state.Last.Code {
			state.Pending = &update
			state.Pending.Sequence = state.Sequence + 1
			if err := r.Journal.Save(*state); err != nil {
				return err
			}
			if control, err = r.flush(ctx, state); err != nil {
				return err
			}
			if terminal(control.State) {
				return r.captureFinal(state, "")
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if control, err = r.control(ctx, state); err != nil {
			return err
		}
		if terminal(control.State) {
			return r.captureFinal(state, "")
		}
	}
}

func (r *Runner) control(ctx context.Context, s *State) (recording.CompanionControl, error) {
	c, err := r.Remote.Control(ctx, s.ID, s.Instance, s.DeviceToken)
	if err != nil {
		return c, err
	}
	if !c.DeadlineAt.Equal(s.Packet.Control.DeadlineAt) || c.Sequence != s.Sequence && (s.Pending == nil || c.Sequence != s.Pending.Sequence) || c.Action != "" && c.Action != "stop" && c.Action != "cancel" || !terminal(c.State) && c.State != "starting" && c.State != "recording" && c.State != "stopping" {
		return c, ErrAuthority
	}
	return c, nil
}

func (r *Runner) flush(ctx context.Context, s *State) (recording.CompanionControl, error) {
	c, err := r.Remote.Submit(ctx, s.ID, s.Instance, s.DeviceToken, *s.Pending)
	if err != nil {
		return c, err
	}
	if c.Sequence != s.Pending.Sequence || !c.DeadlineAt.Equal(s.Packet.Control.DeadlineAt) {
		return c, ErrAuthority
	}
	s.Sequence, s.Last = s.Pending.Sequence, *s.Pending
	s.Pending = nil
	if err := r.Journal.Save(*s); err != nil {
		return c, err
	}
	return c, nil
}

func recordedSnapshot(snapshot recording.Snapshot, failure string) recording.CompanionSnapshot {
	value := recording.CompanionSnapshot{State: snapshot.State, Code: snapshot.Code}
	if failure != "" || value.State != "recording" && value.State != "ready" && value.State != "failed" {
		value.State = "failed"
	}
	if len(value.Code) > recording.ContentLimit || !utf8.ValidString(value.Code) {
		value.State, value.Code, failure = "failed", "", "capture"
	}
	if value.State == "failed" {
		if failure == "" {
			failure = "recorder"
		}
		value.Failure = failure
	}
	return value
}

func (r *Runner) captureFinal(s *State, failure string) error {
	cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	snapshot, err := r.Driver.Stop(cleanup, s.ID)
	if err != nil {
		return errors.New("the owned recorder has not confirmed shutdown; local cleanup remains pending")
	}
	value := recordedSnapshot(snapshot, failure)
	if value.Code == "" {
		value.Code = s.Last.Code
		if s.Pending != nil && s.Pending.Code != "" {
			value.Code = s.Pending.Code
		}
	}
	s.Final = &value
	return r.Journal.Save(*s)
}

func (r *Runner) sendFinal(ctx context.Context, s *State) error {
	if s.Pending != nil {
		c, err := r.flush(ctx, s)
		if err != nil {
			return err
		}
		if terminal(c.State) {
			return nil
		}
	}
	if s.Last.State == s.Final.State && s.Last.Code == s.Final.Code && s.Last.Failure == s.Final.Failure {
		return nil
	}
	value := *s.Final
	value.Sequence = s.Sequence + 1
	s.Pending = &value
	if err := r.Journal.Save(*s); err != nil {
		return err
	}
	_, err := r.flush(ctx, s)
	return err
}
