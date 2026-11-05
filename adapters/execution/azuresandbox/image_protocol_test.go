package azuresandbox

import (
	"net/http"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func TestSandboxImageImportBindsReportedDefinitionAndReadyStates(t *testing.T) {
	for _, plan := range []*ImagePlan{nil, {}} {
		if plan.Verify(DiskImage{}) != hostedexecution.ErrOwnership {
			t.Fatal("incomplete plan admitted an image")
		}
	}
	for _, mode := range []string{"ready", "succeeded", "labels", "dockerfile", "command", "base"} {
		t.Run(mode, func(t *testing.T) {
			p := fixturePlan(t, "go", "discover")
			plan := fixtureImagePlan(t, p)
			client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(r *http.Request) (*http.Response, error) {
				body := readBody(t, r)
				if body["managedIdentityResourceId"] != plan.proof.PullIdentity || body["registryCredentials"] != nil || body["dockerfileContent"] != nil || hash(body["labels"]) != hash(plan.definition()["labels"]) {
					t.Fatal("import request changed authority or ownership")
				}
				delete(body, "managedIdentityResourceId")
				body["id"] = "converted-image"
				state := "Ready"
				if mode == "succeeded" {
					state = "Succeeded"
				}
				body["status"] = map[string]string{"state": state}
				switch mode {
				case "labels":
					delete(body["labels"].(map[string]any), "teststudio-owner")
				case "dockerfile":
					body["dockerfileContent"] = "FROM another-image"
				case "command":
					body["image"].(map[string]any)["cmd"] = []string{"untrusted"}
				case "base":
					body["image"].(map[string]any)["base"] = "public.example/source:latest"
				}
				return jsonResponse(body), nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			image, err := client.ImportImage(t.Context(), plan)
			if mode != "ready" && mode != "succeeded" {
				if err != hostedexecution.ErrOwnership {
					t.Fatal("changed conversion admitted", err)
				}
				return
			}
			if err != nil || !readyImage(image.State) {
				t.Fatal("completed conversion rejected", err)
			}
			job := hostedexecution.Job{Reference: p.proof.Reference, Image: p.proof.Image, Runner: p.proof.Runner, Operation: "discover", ExchangeOrigin: p.config.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: p.proof.Reference.ID, Token: p.token, Deadline: p.proof.Deadline}}
			if _, err := NewPlan(p.config, job, image, plan.proof.Image, time.Now()); err != nil {
				t.Fatal("completed image cannot launch", err)
			}
		})
	}
}
