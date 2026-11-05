package oci

import (
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestOCIInspectionRetainsVersionedSandboxCapability(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		f := fresh(t, runner, false, func(config *configuration, _ *manifest) {
			config.Config.Labels["teststudio.sandbox-supervisor"] = hostedexecution.SandboxSupervisor
		})
		image, err := f.client.InspectImage(t.Context(), f.reference)
		if err != nil || image.Sandbox != hostedexecution.SandboxSupervisor || image.Runner != runner {
			t.Fatal("versioned sandbox capability disappeared from approved source inspection", err)
		}
	}
}
