package workerexchange_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/suitespec"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func TestEnvironmentDiscoveryRequiresFrozenInspectedCapabilities(t *testing.T) {
	for _, mode := range []string{"approved", "missing transport", "missing API policy"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t)
			ref := f.ref
			ref.Installation, ref.Profile, ref.Test = "preparation-fixture", "go-kubernetes-discovery-v1", "catalog"
			input := workerprotocol.DiscoveryInput{Project: ref.Project, Suite: "endpoint-checks", Runner: "go", Image: f.input.Artifact.Image, SourceDigest: "sha256:" + strings.Repeat("c", 64), Environment: mode != "missing transport", RequestPolicy: mode != "missing API policy"}
			ticket, err := f.service.PrepareDiscovery(t.Context(), ref, input, time.Now().Add(30*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			if err := f.service.Activate(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if _, err := f.service.Input(t.Context(), ref.ID, ticket.Token, "discovery-pod"); err != nil {
				t.Fatal(err)
			}
			test := catalog.Test{ID: "API-STATUS", Title: "Read the status endpoint", Category: "API", Type: "api", Runner: "go", State: "enabled", Safety: "safe", Rationale: "Reads the status endpoint only.", TimeoutSeconds: 5, Source: "checks/status_test.go", Selector: "./checks::TestStatus", RequiredBindings: []string{"reader"}}
			manifest := suitespec.Manifest{Schema: 1, Suite: input.Suite, Runner: "go", Type: "api", ModuleRoot: ".", Catalog: "catalog.json", BuildProfile: "go-environment", Tests: suitespec.Tests{Packages: []string{"./checks"}}, RequiredBindings: []string{"reader"}, Browser: &browserpolicy.Spec{Endpoint: "api", Requests: &apipolicy.Spec{Authentication: &apipolicy.Authentication{Binding: "reader", Scheme: "bearer"}, Rules: []apipolicy.Rule{{Method: "GET", PathPrefix: "/status", ReadOnly: true}}, MaxRequests: 2, MaxResponseBytes: 4096}}}
			report := suitespec.Discovery{Schema: suitespec.DiscoverySchema, Manifest: manifest, Origin: suitespec.Origin{SnapshotDigest: input.SourceDigest}, Catalog: catalog.Document{Schema: catalog.SchemaVersion, Tests: []catalog.Test{test}}, Tests: []suitespec.DiscoveredTest{{ID: test.ID, Selector: test.Selector, File: test.Source, Line: 1}}}
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			err = f.service.Submit(t.Context(), ref.ID, ticket.Token, "discovery-pod", workerexchange.Submission{Discovery: &workerprotocol.DiscoveryReport{Raw: raw}})
			if mode == "approved" {
				if err != nil {
					t.Fatal("approved environment discovery rejected", err)
				}
			} else if !errors.Is(err, workerexchange.ErrInvalid) {
				t.Fatal("discovery manufactured an uninspected capability", err)
			}
			changed := input
			changed.Environment = !input.Environment
			if _, err := f.service.PrepareDiscovery(t.Context(), ref, changed, ticket.Deadline); !errors.Is(err, workerexchange.ErrConflict) {
				t.Fatal("prepared discovery capabilities changed", err)
			}
		})
	}
}
