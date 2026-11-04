package jobsupervisor

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
)

func (p *Process) runCommand(ctx context.Context, wire []byte, discovery bool, onLine func(string)) (stdout, stderr execution.Output, exit int, result error) {
	exit = -1
	command, err := p.command(ctx, discovery)
	if err != nil || command == nil {
		return stdout, stderr, exit, ErrRunner
	}
	command.Dir = p.Directory
	command.Env = childEnvironment()
	command.Stdin = bytes.NewReader(wire)
	ownGroup(command)
	grace := p.StopGrace
	if grace == 0 {
		grace = 7 * time.Second
	}
	if grace < 0 || grace > 10*time.Second {
		return stdout, stderr, exit, ErrRunner
	}
	var forced *time.Timer
	var forceDone chan struct{}
	var timerMu sync.Mutex
	command.Cancel = func() error {
		timerMu.Lock()
		defer timerMu.Unlock()
		err := stopGroup(command, false)
		forceDone = make(chan struct{})
		forced = time.AfterFunc(grace, func() { _ = stopGroup(command, true); close(forceDone) })
		return err
	}
	command.WaitDelay = grace + 2*time.Second
	stdout.OnLine = onLine
	command.Stdout, command.Stderr = &stdout, &stderr
	result = command.Run()
	timerMu.Lock()
	waitForForce := forced != nil && !forced.Stop()
	timerMu.Unlock()
	if waitForForce {
		<-forceDone
	}
	if command.ProcessState != nil {
		exit = command.ProcessState.ExitCode()
	}
	if command.Process != nil {
		_ = stopGroup(command, true)
	}
	return
}
