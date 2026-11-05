package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestSandboxCommandClientFixesCommandAndRequiresAnExplicitExitResult(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+"/"+operation, func(t *testing.T) {
				p := fixturePlan(t, runner, operation)
				client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(r *http.Request) (*http.Response, error) {
					if r.Method != "POST" || r.URL.Host != "management.westus2.azuredevcompute.io" || r.URL.Path != p.config.path()+"/sandboxes/native-sandbox/executeShellCommand" || r.GetBody != nil {
						t.Fatal("command request escaped the fixed nonreplayable endpoint")
					}
					if at, ok := r.Context().Deadline(); !ok || !at.Equal(p.proof.Deadline) {
						t.Fatal("command request inherited a short polling timeout")
					}
					body := readBody(t, r)
					if body["workingDirectory"] != "/suite" || body["command"] != p.commandBody()["command"] || len(body) != 2 {
						t.Fatal("command request contains overrides")
					}
					text, _ := json.Marshal(body)
					if strings.Contains(string(text), secretTicket) || !strings.Contains(body["command"].(string), "--sandbox-command") {
						t.Fatal("command request exposed credentials or omitted identity enforcement")
					}
					return response(200, "{\"exitCode\":0,\"stdout\":\"private-command-canary\",\"stderr\":\"\"}"), nil
				}))
				if err != nil {
					t.Fatal(err)
				}
				defer client.Close()
				result, err := client.Execute(t.Context(), p, "native-sandbox")
				data, _ := json.Marshal(result)
				if err != nil || result.ExitCode != 0 || strings.Contains(string(data), "private-command-canary") {
					t.Fatal("explicit command completion was lost or private output exposed", err)
				}
			})
		}
	}
	for _, body := range []string{"{}", "{\"exitCode\":null}", "{\"exitCode\":\"0\"}", "{\"exitCode\":256}", "{\"exitCode\":-1}", "{\"exitCode\":0.5}"} {
		p := fixturePlan(t, "go", "")
		client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(*http.Request) (*http.Response, error) { return response(200, body), nil }))
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Execute(context.Background(), p, "native-sandbox")
		client.Close()
		if err != hostedexecution.ErrUnavailable {
			t.Fatal("missing or invalid exit code became success", err)
		}
	}
}

func TestSandboxCommandClientRetainsUncertaintyAndRejectsExpiredWork(t *testing.T) {
	p := fixturePlan(t, "go", "")
	count := 0
	client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(*http.Request) (*http.Response, error) {
		count++
		value := response(302, "private-command-canary")
		value.Header.Set("Location", "https://unrelated.example/collect")
		return value, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Execute(t.Context(), p, "native-sandbox"); err != hostedexecution.ErrUnavailable || count != 1 {
		t.Fatal("redirected command was retried or became known", err)
	}
	p.proof.Deadline = time.Now().UTC().Add(-time.Second)
	p, err = newPlan(p.config, p.proof)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Execute(t.Context(), p, "native-sandbox"); err != hostedexecution.ErrRejected || count != 1 {
		t.Fatal("expired work reached the command transport", err)
	}
}
