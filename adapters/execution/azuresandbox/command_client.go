package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type CommandResult struct{ ExitCode int }

func shellWord(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

func (p *Plan) commandBody() map[string]any {
	args := []string{"/runner/teststudio-job", "--sandbox-command", "--runner=" + p.proof.Runner, "--exchange-origin=" + p.config.ExchangeOrigin}
	if p.proof.Operation == "discover" {
		args = append(args, "--operation=discover")
	}
	for index := range args {
		args[index] = shellWord(args[index])
	}
	return map[string]any{"command": "exec " + strings.Join(args, " "), "workingDirectory": "/suite"}
}

func (p *Plan) commandHash() string { return hash(p.commandBody()) }

func (c *Client) Execute(ctx context.Context, plan *Plan, id string) (CommandResult, error) {
	if plan == nil || hash(plan.config) != hash(c.config) || !identifier.MatchString(id) {
		return CommandResult{}, hostedexecution.ErrOwnership
	}
	if _, err := Restore(c.config, plan.Proof()); err != nil {
		return CommandResult{}, err
	}
	remaining := time.Until(plan.proof.Deadline)
	if remaining <= 0 || remaining > 2*time.Hour {
		return CommandResult{}, hostedexecution.ErrRejected
	}
	ctx, cancel := context.WithDeadline(ctx, plan.proof.Deadline)
	defer cancel()
	data, err := c.send(ctx, c.commandHTTP, http.MethodPost, c.target("/sandboxes/"+id+"/executeShellCommand"), plan.commandBody())
	if err != nil {
		return CommandResult{}, err
	}
	var result struct{ ExitCode *int }
	if ctx.Err() != nil || json.Unmarshal(data, &result) != nil || result.ExitCode == nil || *result.ExitCode < 0 || *result.ExitCode > 255 {
		return CommandResult{}, hostedexecution.ErrUnavailable
	}
	return CommandResult{ExitCode: *result.ExitCode}, nil
}
