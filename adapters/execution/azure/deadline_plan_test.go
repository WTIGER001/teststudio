package azureworker

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func deadlineDefinition(c DeadlineConfig, s DeadlineSpec) l.Workflow {
	d := c.definition(s)
	d.ID, d.Name, d.Type = to.Ptr(c.resourceID(s)), to.Ptr(c.name(s)), to.Ptr("Microsoft.Logic/workflows")
	d.Properties.ProvisioningState = to.Ptr(l.WorkflowProvisioningStateSucceeded)
	d.Properties.AccessEndpoint = to.Ptr("https://provider-metadata.invalid")
	d.Identity.TenantID = to.Ptr("readonly-tenant")
	d.Identity.UserAssignedIdentities[c.StopIdentity].ClientID = to.Ptr("readonly-client")
	return d
}

func TestAzureDeadlinePlanBindsAbsoluteStopAndSeparateIdentity(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+operation, func(t *testing.T) {
				worker, job, plan, _ := fixture(t, runner, operation)
				c := DeadlineConfig{Worker: worker, ResourceGroup: "deadlines", StopIdentity: worker.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "deadline-stop")}
				s := plan.deadlineSpec()
				if c.validate() != nil || c.validateSpec(s) != nil || c.verify(s, deadlineDefinition(c, s), true) != nil {
					t.Fatal("valid owned deadline rejected")
				}
				d := document(c.definition(s))
				props := d["properties"].(map[string]any)
				def := props["definition"].(map[string]any)
				trigger := def["triggers"].(map[string]any)["Deadline"].(map[string]any)
				recurrence := trigger["recurrence"].(map[string]any)
				at, err := time.Parse(time.RFC3339, recurrence["startTime"].(string))
				if err != nil || at.Before(s.At) || at.Sub(s.At) >= time.Second || recurrence["frequency"] != "Minute" || recurrence["interval"] != float64(1) {
					t.Fatal("deadline ran early or lost its absolute start")
				}
				action := def["actions"].(map[string]any)["Stop"].(map[string]any)
				inputs := action["inputs"].(map[string]any)
				auth := inputs["authentication"].(map[string]any)
				serialized, _ := json.Marshal(d)
				if inputs["method"] != "POST" || inputs["uri"] != "https://management.azure.com"+s.ResourceID+"/stop?api-version=2025-07-01" || auth["identity"] != c.StopIdentity || auth["audience"] != "https://management.azure.com/" || action["operationOptions"] != "DisableAsyncPattern" || strings.Contains(string(serialized), job.Ticket.Token) {
					t.Fatal("timer gained another action or credential")
				}
				bad := c
				bad.StopIdentity = worker.PullIdentity
				if bad.validate() == nil {
					t.Fatal("timer reused the untrusted worker identity")
				}
				bad = c
				bad.ResourceGroup = worker.ResourceGroup
				if bad.validate() == nil {
					t.Fatal("timer shared the worker namespace")
				}
				s.ResourceID += "/executions/foreign"
				requireDeadlineError(t, c.validateSpec(s), hostedexecution.ErrOwnership)
			})
		}
	}
}

func TestAzureDeadlinePlanRejectsWritableDrift(t *testing.T) {
	worker, _, plan, _ := fixture(t, "go", "")
	c := DeadlineConfig{Worker: worker, ResourceGroup: "deadlines", StopIdentity: worker.resourceID("Microsoft.ManagedIdentity", "userAssignedIdentities", "deadline-stop")}
	s := plan.deadlineSpec()
	cases := map[string]func(*l.Workflow){
		"identity":         func(d *l.Workflow) { d.Identity.Type = to.Ptr(l.ManagedServiceIdentityTypeSystemAssigned) },
		"disabled":         func(d *l.Workflow) { d.Properties.State = to.Ptr(l.WorkflowStateDisabled) },
		"specification":    func(d *l.Workflow) { d.Tags["teststudio-spec"] = to.Ptr(strings.Repeat("f", 64)) },
		"foreign-resource": func(d *l.Workflow) { d.ID = to.Ptr("foreign") },
		"parameter": func(d *l.Workflow) {
			d.Properties.Parameters = map[string]*l.WorkflowParameter{"injected": {Value: "foreign"}}
		},
		"action": func(d *l.Workflow) {
			def := d.Properties.Definition.(map[string]any)
			def["actions"].(map[string]any)["Injected"] = map[string]any{"type": "Http"}
		},
		"early-time": func(d *l.Workflow) {
			d.Properties.Definition.(map[string]any)["triggers"].(map[string]any)["Deadline"].(map[string]any)["recurrence"].(map[string]any)["startTime"] = "2000-01-01T00:00:00Z"
		},
		"endpoint": func(d *l.Workflow) {
			d.Properties.Definition.(map[string]any)["actions"].(map[string]any)["Stop"].(map[string]any)["inputs"].(map[string]any)["uri"] = "https://foreign.invalid"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := deadlineDefinition(c, s)
			mutate(&d)
			requireDeadlineError(t, c.verify(s, d, true), hostedexecution.ErrOwnership)
		})
	}
	d := deadlineDefinition(c, s)
	d.Properties.ProvisioningState = to.Ptr(l.WorkflowProvisioningStateInProgress)
	requireDeadlineError(t, c.verify(s, d, true), hostedexecution.ErrUnavailable)
	if c.verify(s, d, false) != nil {
		t.Fatal("cleanup required successful provisioning")
	}
}
