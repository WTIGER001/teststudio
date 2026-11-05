package lifecycle

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func gatewayWindow(t *testing.T, identifier string, duration time.Duration) {
	t.Helper()
	origin := os.Getenv("TESTSTUDIO_API_BASE_URL")
	if origin == "" {
		t.Fatal("The reviewed gateway relay is required")
	}
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Get(origin + "/api/private/status")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]string
	err = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&result)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || result["status"] != "ready" {
		t.Fatal("The gateway did not reach the authenticated fixture")
	}
	marker, _ := json.Marshal(map[string]any{"testId": identifier, "connected": true})
	if err := os.WriteFile("/output/gateway-lifecycle-ready.json", marker, 0600); err != nil {
		t.Fatal("The bounded lifecycle marker is unavailable")
	}
	t.Log("The gateway reached the authenticated fixture; the lifecycle window is open")
	time.Sleep(duration)
}

func TestGatewayCancellation(t *testing.T) {
	gatewayWindow(t, "NET-GATEWAY-CANCEL-001", 40*time.Second)
}

func TestGatewayRecovery(t *testing.T) {
	gatewayWindow(t, "NET-GATEWAY-RECOVERY-001", 12*time.Second)
}
