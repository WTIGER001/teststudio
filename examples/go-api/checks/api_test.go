package checks

import (
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestServiceHealth(t *testing.T) {
	base := os.Getenv("TESTSTUDIO_API_BASE_URL")
	if base == "" {
		t.Fatal("Run this API test through TestStudio with a registered environment")
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	t.Cleanup(client.CloseIdleConnections)
	t.Run("Read the reviewed health endpoint", func(t *testing.T) {
		response, err := client.Get(base + "/health")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(io.LimitReader(response.Body, 65537))
		if err != nil || len(body) > 65536 {
			t.Fatal("Response exceeds the reviewed diagnostic limit")
		}
		if response.StatusCode != 200 {
			t.Fatalf("Expected status 200, received %d: %s", response.StatusCode, body)
		}
		t.Logf("Health response: status %d, %d bytes", response.StatusCode, len(body))
	})
}
