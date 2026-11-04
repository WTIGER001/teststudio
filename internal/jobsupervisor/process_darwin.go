package jobsupervisor

import (
	"os/exec"
	"syscall"
)

func ownGroup(command *exec.Cmd) { command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func stopGroup(command *exec.Cmd, force bool) error {
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-command.Process.Pid, signal)
}
