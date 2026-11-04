package localcontainer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/catalog"
)

func TestAPIPolicyRequiresCapableGatewayAndRunnerImages(t *testing.T) {
	directory := t.TempDir()
	metadata := filepath.Join(directory, "capability.json")
	binary := filepath.Join(directory, "docker")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\ncat \"$TESTSTUDIO_API_LABELS\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TESTSTUDIO_API_LABELS", metadata)
	driver := &Executor{Docker: binary, GatewayImage: "sha256:" + strings.Repeat("a", 64)}
	artifact := catalog.Artifact{Image: "sha256:" + strings.Repeat("b", 64)}
	for _, labels := range []string{`{}`, `{"teststudio.api-policy":"legacy"}`, `{"teststudio.api-policy":"request-policy/v1"}`, `{"teststudio.api-policy":"request-policy/v1","teststudio.work-item-bytes":"4194304"}`} {
		if err := os.WriteFile(metadata, []byte(labels), 0600); err != nil {
			t.Fatal(err)
		}
		err := driver.admitAPIPolicy(t.Context(), artifact)
		if (err == nil) != strings.Contains(labels, "4194304") {
			t.Fatal("unsupported image capability is admitted", labels, err)
		}
	}
	for _, labels := range []string{`{}`, `{"teststudio.api-credentials":"gateway-injection/v1"}`} {
		if err := os.WriteFile(metadata, []byte(labels), 0600); err != nil {
			t.Fatal(err)
		}
		err := driver.admitAPICredentials(t.Context())
		if (err == nil) != strings.Contains(labels, "gateway-injection/v1") {
			t.Fatal("credential-less gateway image can claim secret injection")
		}
	}
}
