package runner

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

func command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Cancel = func() error { return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run() }
	cmd.WaitDelay = 2 * time.Second
	return cmd
}
