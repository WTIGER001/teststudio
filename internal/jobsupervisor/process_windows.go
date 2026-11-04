package jobsupervisor

import "os/exec"

func ownGroup(command *exec.Cmd)                    {}
func stopGroup(command *exec.Cmd, force bool) error { return command.Process.Kill() }
