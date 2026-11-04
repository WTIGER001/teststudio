package localcontainer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/apicredential"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/execution"
)

func (e *Executor) startGateway(ctx context.Context, ref execution.Reference, target *browserpolicy.Target, credential *apicredential.Value) error {
	if target == nil {
		return errors.New("environment browser execution requires its reviewed target")
	}
	if err := browserpolicy.ValidateTarget(*target); err != nil {
		return err
	}
	if err := apicredential.Validate(*target, credential); err != nil {
		return err
	}
	if ref.Gateway != e.Reference(ref).Gateway {
		return errors.New("gateway ownership does not match the execution")
	}
	policy, err := json.Marshal(target)
	if err != nil {
		return err
	}
	args := []string{"run", "--detach", "--pull=never", "--name", ref.Gateway, "--network=bridge", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=64", "--memory=128m", "--cpus=1", "--user", "1000:1000", "--env", "TESTSTUDIO_BROWSER_TARGET=" + string(policy), "--mount", "type=volume,src=" + ref.Volume + ",dst=/output"}
	if credential != nil {
		args = append(args, "--tmpfs", "/run/teststudio:rw,nosuid,noexec,size=1048576,uid=1000,gid=1000,mode=0700")
	}
	for key, value := range labels(ref) {
		args = append(args, "--label", key+"="+value)
	}
	args = append(args, e.GatewayImage)
	wall, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	_, err = e.output(wall, args...)
	if err != nil {
		return err
	}
	if credential != nil {
		data, err := json.Marshal(credential)
		if err != nil {
			return errors.New("API credential initialization is unavailable")
		}
		command := exec.CommandContext(wall, e.binary(), "exec", "--interactive", ref.Gateway, "/gateway", "--configure")
		command.Stdin = bytes.NewReader(data)
		if command.Run() != nil {
			return errors.New("API credential initialization is unavailable")
		}
	}
	gatewayRef := ref
	gatewayRef.Container = ref.Gateway
	for wall.Err() == nil {
		info, exists, err := e.container(wall, gatewayRef)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("the selected browser gateway disappeared before dispatch")
		}
		if !info.State.Running || info.State.Health.Status == "unhealthy" {
			log, _ := e.output(wall, "logs", "--tail", "10", ref.Gateway)
			return fmt.Errorf("the selected browser gateway could not start: %s", strings.TrimSpace(string(log[:min(len(log), 4096)])))
		}
		if info.State.Health.Status == "healthy" {
			return nil
		}
		select {
		case <-wall.Done():
		case <-time.After(100 * time.Millisecond):
		}
	}
	return errors.New("the selected browser gateway did not become ready before dispatch")
}
