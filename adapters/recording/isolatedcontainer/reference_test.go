package isolatedcontainer

import (
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/recordingworker"
)

func TestRecordingNetworkReferencePreservesLegacyCleanupAndRejectsPolicyChanges(t *testing.T) {
	binding := recordingworker.Binding{ID: "session", Actor: "tester", Project: "project", Suite: "browser", URL: "https://website.test"}
	ref := recordingworker.Reference{Provider: "docker-recording/v1", RuntimeImage: "sha256:" + strings.Repeat("a", 64), GatewayImage: "sha256:" + strings.Repeat("b", 64), PolicyHash: SeccompHash}
	bindNames := func() {
		ref.Fingerprint = fingerprint(binding, ref)
		prefix := "ts-recording-" + ref.Fingerprint[:32]
		ref.Container, ref.Gateway, ref.Volume = prefix+"-browser", prefix+"-gateway", prefix+"-output"
	}
	bindNames()
	if !validReference(binding, ref) {
		t.Fatal("legacy recording resources cannot recover owned cleanup")
	}
	legacy := ref.Container
	ref.NetworkHash = strings.Repeat("c", 64)
	bindNames()
	if !validReference(binding, ref) || ref.Container == legacy {
		t.Fatal("new network policy was not bound to separate owned resources")
	}
	ref.NetworkHash = strings.Repeat("d", 64)
	if validReference(binding, ref) {
		t.Fatal("changed network policy retained resource ownership")
	}
}
