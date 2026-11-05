package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/ecsidentity"
)

type metadataTransport func(*http.Request) (*http.Response, error)

func (f metadataTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestECSGatewayBindsMetadataTaskIdentityWithoutForwardingSecrets(t *testing.T) {
	identifier := strings.Repeat("a", 32)
	for _, mode := range []string{"valid", "redirect", "invalid arn", "oversized", "trailing document", "explicit instance"} {
		t.Run(mode, func(t *testing.T) {
			values := map[string]string{"ECS_CONTAINER_METADATA_URI_V4": "http://169.254.170.2/v4/fixture", "TESTSTUDIO_GATEWAY_TOKEN": "private-initialization-canary"}
			if mode == "explicit instance" {
				values["TESTSTUDIO_GATEWAY_INSTANCE"] = "forged"
			}
			requests := 0
			transport := metadataTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.String() != "http://169.254.170.2/v4/fixture/task" || r.Method != "GET" || len(r.Header) != 0 || r.Body != nil {
					t.Fatal("metadata read received a secret or an unexpected endpoint")
				}
				body := `{"TaskARN":"arn:aws:ecs:us-east-1:123456789012:task/studio/` + identifier + `","FutureField":true}`
				status := http.StatusOK
				switch mode {
				case "redirect":
					status = http.StatusFound
				case "invalid arn":
					body = `{"TaskARN":"arn:aws:iam::123456789012:role/not-a-task"}`
				case "oversized":
					body = strings.Repeat("x", (64<<10)+1)
				case "trailing document":
					body += `{}`
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Location": {"http://outside.test/credential-canary"}}}, nil
			})
			lookup := func(name string) (string, bool) { value, ok := values[name]; return value, ok }
			bound, err := ecsGatewayLookup(context.Background(), lookup, transport)
			if mode == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				if instance, found := bound("TESTSTUDIO_GATEWAY_INSTANCE"); !found || instance != identifier {
					t.Fatal("metadata task identity was not used for private initialization")
				}
				if token, _ := bound("TESTSTUDIO_GATEWAY_TOKEN"); token != values["TESTSTUDIO_GATEWAY_TOKEN"] {
					t.Fatal("scoped initialization ticket changed")
				}
			} else if err != ecsidentity.ErrIdentity {
				t.Fatal("invalid metadata was accepted", mode, err)
			}
			if requests > 1 || mode == "explicit instance" && requests != 0 {
				t.Fatal("metadata followed another endpoint or accepted an explicit instance")
			}
		})
	}
}

func TestECSGatewayRejectsMetadataEndpointOverridesBeforeNetwork(t *testing.T) {
	for _, raw := range []string{"", "https://169.254.170.2/v4/fixture", "http://169.254.169.254/v4/fixture", "http://outside.test/v4/fixture", "http://169.254.170.2/credentials", "http://169.254.170.2/v4/fixture?token=canary", "http://user:canary@169.254.170.2/v4/fixture", "http://169.254.170.2/v4/fixture/../other", "http://169.254.170.2/v4/%66ixture", "http://169.254.170.2:80/v4/fixture"} {
		lookup := func(name string) (string, bool) { return raw, name == "ECS_CONTAINER_METADATA_URI_V4" }
		_, err := ecsGatewayLookup(t.Context(), lookup, metadataTransport(func(*http.Request) (*http.Response, error) {
			t.Fatal("an unapproved metadata endpoint received a request")
			return nil, nil
		}))
		if err != ecsidentity.ErrIdentity {
			t.Fatal("unapproved metadata endpoint was accepted")
		}
	}
}
