package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (c *Controller) Close() {
	c.commandMutex.Lock()
	c.closed = true
	if c.commandCancel != nil {
		c.commandCancel()
	}
	c.commandMutex.Unlock()
	c.commandGroup.Wait()
}

func (c *Controller) cancelCommand(id string) {
	c.commandMutex.Lock()
	cancel := c.activeCommands[id]
	c.commandMutex.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Controller) reserveCommand(ctx context.Context, ref execution.Reference) (*Plan, string, error) {
	for range 8 {
		d, row, plan, err := c.load(ctx, ref)
		if err != nil {
			return nil, "", err
		}
		if d.CommandAttempted {
			return nil, "", nil
		}
		if d.Stopping || d.Removed || d.Quiesced || d.Receipt == "" || d.LastState != "Running" {
			return nil, "", hostedexecution.ErrUnavailable
		}
		work, guard, err := c.transport(ctx, d)
		if err != nil {
			return nil, "", err
		}
		if !work.DispatchReady || work.Revoked || work.Instance != "" || work.Submission != nil || !c.now().Before(work.Deadline) {
			return nil, "", hostedexecution.ErrUnavailable
		}
		var access workerstate.Access
		if err := bounded(ctx, func(ctx context.Context) error {
			var err error
			access, err = c.exchange.Authorize(ctx, ref)
			return err
		}); err != nil || access.Deadline.Before(work.Deadline) || !c.now().Before(access.Deadline) {
			return nil, "", hostedexecution.ErrUnavailable
		}
		d.CommandAttempted = true
		d.CommandHash = plan.commandHash()
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, d)
		if err != nil {
			return nil, "", hostedexecution.ErrUnavailable
		}
		changes := []storage.Change{change, {Record: guard, Assert: true}}
		for _, record := range access.Records {
			changes = append(changes, storage.Change{Record: record, Assert: true})
		}
		err = c.store.Commit(ctx, changes...)
		if err == storage.ErrConflict {
			continue
		}
		if err != nil {
			return nil, "", hostedexecution.ErrUnavailable
		}
		return plan, d.Receipt, nil
	}
	return nil, "", hostedexecution.ErrUnavailable
}

func (c *Controller) StartCommand(ctx context.Context, ref execution.Reference) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	if c.commands == nil {
		return hostedexecution.ErrConfiguration
	}
	d, _, _, err := c.load(ctx, ref)
	if err != nil {
		return err
	}
	if d.CommandAttempted {
		return nil
	}
	state, err := c.Observe(ctx, ref)
	if err != nil {
		return err
	}
	if state.AdmissionPending || state.Quiesced || state.Removed {
		return hostedexecution.ErrUnavailable
	}
	c.commandMutex.Lock()
	defer c.commandMutex.Unlock()
	if c.closed {
		return hostedexecution.ErrUnavailable
	}
	plan, id, err := c.reserveCommand(ctx, ref)
	if err != nil || plan == nil {
		return err
	}
	wall, cancel := context.WithDeadline(c.commandContext, plan.proof.Deadline)
	c.activeCommands[ref.ID] = cancel
	c.commandGroup.Add(1)
	go c.runCommand(wall, cancel, plan, id)
	return nil
}

func (c *Controller) runCommand(ctx context.Context, cancel context.CancelFunc, plan *Plan, id string) {
	ref := plan.proof.Reference
	defer c.commandGroup.Done()
	defer func() { cancel(); c.commandMutex.Lock(); delete(c.activeCommands, ref.ID); c.commandMutex.Unlock() }()
	result, err := c.commands.Execute(ctx, plan, id)
	if ctx.Err() != nil || !c.now().Before(plan.proof.Deadline) || result.ExitCode < 0 || result.ExitCode > 255 {
		err = hostedexecution.ErrUnavailable
	}
	finish, release := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer release()
	_, saveErr := c.update(finish, ref, false, func(d *dispatch) error {
		if !d.CommandAttempted || d.CommandHash != plan.commandHash() || d.Receipt != id || d.CommandExitCode != nil || d.CommandUnknown || d.CommandRejected {
			return hostedexecution.ErrOwnership
		}
		if err != nil {
			d.CommandRejected = controlError(err) == hostedexecution.ErrRejected
			d.CommandUnknown = !d.CommandRejected
			return nil
		}
		code, at := result.ExitCode, c.now().UTC()
		d.CommandExitCode = &code
		d.CommandObservedAt = &at
		return nil
	})
	if saveErr == nil {
		_ = c.Stop(finish, ref)
	}
}

func (c *Controller) Status(ctx context.Context, ref execution.Reference) (status hostedexecution.Status, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, _, _, err := c.load(ctx, ref)
	if err != nil {
		return status, err
	}
	status = hostedexecution.Status{Reference: ref, Image: d.Proof.Image, Deadline: d.Proof.Deadline, Phase: "pending", AdmissionPending: true}
	if d.CreateRejected || d.CreationBlocked {
		return status, hostedexecution.ErrNotFound
	}
	if !d.Removed {
		state, err := c.Observe(ctx, ref)
		if err != nil {
			return status, err
		}
		if state.Instance.State == "Running" {
			status.Phase = "running"
		}
		status.AdmissionPending = state.AdmissionPending
		work, _, err := c.transport(ctx, d)
		if err != nil {
			return status, err
		}
		if !state.AdmissionPending && !state.Quiesced && !d.Stopping && !d.CommandAttempted && work.DispatchReady && !work.Revoked {
			if err := c.StartCommand(ctx, ref); err != nil {
				return status, err
			}
		}
		d, _, _, err = c.load(ctx, ref)
		if err != nil {
			return status, err
		}
	}
	if d.CommandAttempted && !d.Quiesced && (d.CommandExitCode != nil || d.CommandUnknown || d.CommandRejected) {
		if err := c.Stop(ctx, ref); err == hostedexecution.ErrOwnership {
			return status, err
		}
		d, _, _, err = c.load(ctx, ref)
		if err != nil {
			return status, err
		}
	}
	if d.Stopping {
		status.AdmissionPending = false
	}
	if d.Quiesced {
		status.AdmissionPending = false
		status.Phase = "stopped"
		if d.CommandExitCode != nil && !d.CommandSuppressed {
			code := *d.CommandExitCode
			status.ExitCode = &code
			status.Phase = "failed"
			if code == 0 {
				status.Phase = "succeeded"
			}
		}
	}
	return status, nil
}
