package azuresandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxPlansFreezeIsolatedLaunchAndTicketFreeRecovery(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"/"+operation, func(t *testing.T) {
				p := fixturePlan(t, runner, operation)
				body, err := p.launch(time.Now())
				if err != nil {
					t.Fatal(err)
				}
				policy := body["egressPolicy"].(map[string]any)
				if policy["defaultAction"] != "Deny" || policy["trafficInspection"] != "Full" || body["skipEgressProxy"] != false || body["customerVnetConnectionName"] != p.config.VNetConnection {
					t.Fatal("launch lacks restricted private exchange access")
				}
				if env := body["environment"].(map[string]string); len(env) != 2 || env["TESTSTUDIO_JOB_TOKEN"] != secretTicket || env["TESTSTUDIO_JOB_EXECUTION"] != p.proof.Reference.ID {
					t.Fatal("unexpected worker credential environment")
				}
				proof := p.Proof()
				data, _ := json.Marshal(proof)
				diagnostics, _ := json.Marshal(p)
				if strings.Contains(string(data)+string(diagnostics)+fmt.Sprintf("%+v %#v %+v %#v", p, p, *p, *p), secretTicket) {
					t.Fatal("launch credential escaped into saved proof or diagnostics")
				}
				restored, err := Restore(p.config, proof)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := restored.launch(time.Now()); !errors.Is(err, hostedexecution.ErrRejected) {
					t.Fatal("saved proof permits another dispatch")
				}
				if _, err := p.launch(proof.Deadline); !errors.Is(err, hostedexecution.ErrRejected) {
					t.Fatal("deadline permits dispatch")
				}
				actualBody := instanceBody(t, p, "sandbox-123")
				actualBody["connections"] = []any{}
				actualBody["ports"] = []any{}
				actualBody["createdAt"] = time.Now().UTC().Format(time.RFC3339Nano)
				delete(actualBody, "cmd")
				data, _ = json.Marshal(actualBody)
				actual, err := snapshot(data)
				if err != nil || restored.Verify(actual) != nil {
					t.Fatal("matching sandbox cannot be reconciled", err)
				}
				data, _ = json.Marshal(actual)
				if strings.Contains(string(data), secretTicket) {
					t.Fatal("snapshot retains credential")
				}
			})
		}
	}
}

func TestSandboxResponseChangesCannotAdoptAnUnownedExecution(t *testing.T) {
	p := fixturePlan(t, "playwright", "")
	changes := map[string]func(map[string]any){
		"command": func(m map[string]any) { m["entrypoint"] = []string{"/bin/sh"} },
		"public-image": func(m map[string]any) {
			m["sourcesRef"] = map[string]any{"diskImage": map[string]any{"name": "ubuntu", "isPublic": true}}
		},
		"new-ticket": func(m map[string]any) {
			m["environment"].(map[string]any)["TESTSTUDIO_JOB_TOKEN"] = strings.Repeat("x", 43)
		},
		"extra-credential": func(m map[string]any) { m["environment"].(map[string]any)["AZURE_CLIENT_SECRET"] = "credential-canary" },
		"proxy-bypass":     func(m map[string]any) { m["skipEgressProxy"] = true },
		"allow-egress":     func(m map[string]any) { m["egressPolicy"].(map[string]any)["defaultAction"] = "Allow" },
		"shared-volume":    func(m map[string]any) { m["volumes"] = []any{map[string]any{"id": "shared-data"}} },
		"foreign-network":  func(m map[string]any) { m["customerVnetConnectionName"] = "other-network" },
		"snapshot": func(m map[string]any) {
			m["sourcesRef"] = map[string]any{"snapshot": map[string]any{"id": "old-execution"}}
		},
		"exposed-port":            func(m map[string]any) { m["ports"] = []any{map[string]any{"port": 8080}} },
		"foreign-owner":           func(m map[string]any) { m["labels"].(map[string]any)["teststudio-owner"] = strings.Repeat("d", 64) },
		"resource-class":          func(m map[string]any) { m["resources"].(map[string]any)["cpu"] = "1000m" },
		"unknown-setting":         func(m map[string]any) { m["futureExecutionSetting"] = true },
		"virtualization-override": func(m map[string]any) { m["vmmType"] = "unexpected-runtime" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			body := instanceBody(t, p, "sandbox-123")
			change(body)
			data, _ := json.Marshal(body)
			actual, err := snapshot(data)
			if err != nil || !errors.Is(p.Verify(actual), hostedexecution.ErrOwnership) {
				t.Fatal("changed execution accepted", err)
			}
		})
	}
	proof := p.Proof()
	proof.DiskImage = "other-image"
	if _, err := Restore(p.config, proof); !errors.Is(err, hostedexecution.ErrOwnership) {
		t.Fatal("changed recovery proof accepted")
	}
	c := p.config
	before, _ := ExecutionProfile(c)
	c.NetworkProfile = strings.Repeat("b", 64)
	after, _ := ExecutionProfile(c)
	if before == after {
		t.Fatal("timing profile does not bind network policy")
	}
}

func TestSandboxConfigurationRejectsUnscopedEndpointsAndImages(t *testing.T) {
	c := fixtureConfig()
	for _, origin := range []string{"http://exchange.studio.example", "https://exchange.studio.example:443", "https://exchange.studio.example/", "https://user@exchange.studio.example", "https://127.0.0.1", "https://bad..example", "https://exchange.studio.example?token=hidden"} {
		bad := c
		bad.ExchangeOrigin = origin
		if Validate(bad) == nil {
			t.Fatal("invalid exchange origin accepted")
		}
	}
	for _, image := range []string{c.Registry + "/" + c.RepositoryPrefix + "/test:latest", "other.azurecr.io/" + c.RepositoryPrefix + "/test@sha256:" + strings.Repeat("a", 64), c.Registry + "/other/test@sha256:" + strings.Repeat("a", 64)} {
		if c.validImage(image) {
			t.Fatal("unapproved source image accepted")
		}
	}
}
