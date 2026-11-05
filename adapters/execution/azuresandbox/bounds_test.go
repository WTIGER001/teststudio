package azuresandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type observedBody struct {
	io.Reader
	closed bool
}

func (b *observedBody) Close() error { b.closed = true; return nil }

func TestSandboxResponseLimitsCloseBodiesAndKeepUnknownStatesUnavailable(t *testing.T) {
	p := fixturePlan(t, "go", "")
	unknown := instanceBody(t, p, "sandbox-123")
	unknown["state"] = "FutureSuccess"
	unknownJSON, _ := json.Marshal(unknown)
	for name, value := range map[string]struct {
		body   string
		length int64
	}{
		"invalid-json":      {"not-json " + secretTicket, -1},
		"unknown-state":     {string(unknownJSON), -1},
		"declared-oversize": {"", (4 << 20) + 1},
		"chunked-oversize":  {strings.Repeat("x", (4<<20)+1), -1},
	} {
		t.Run(name, func(t *testing.T) {
			body := &observedBody{Reader: strings.NewReader(value.body)}
			client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: body, ContentLength: value.length}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			_, err = client.Read(context.Background(), "sandbox-123")
			if err != hostedexecution.ErrUnavailable || !body.closed {
				t.Fatal("unbounded response or invented terminal state")
			}
		})
	}
}

func TestSandboxInventoryRejectsDuplicateIDsAndPaginationBeyondTheBound(t *testing.T) {
	p := fixturePlan(t, "go", "")
	for _, duplicate := range []bool{true, false} {
		t.Run(fmt.Sprint("duplicate=", duplicate), func(t *testing.T) {
			calls := 0
			client, err := NewClient(p.config, &credential{t: t}, roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if duplicate {
					return jsonResponse([]any{instanceBody(t, p, "sandbox-123"), instanceBody(t, p, "sandbox-123")}), nil
				}
				return jsonResponse(map[string]any{"value": []any{instanceBody(t, p, fmt.Sprint("sandbox-", calls))}, "nextLink": p.config.endpoint() + p.config.path() + "/sandboxes?api-version=" + APIVersion + "&$skiptoken=" + fmt.Sprint(calls)}), nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			items, err := client.Inventory(context.Background())
			wantCalls := 8
			if duplicate {
				wantCalls = 1
			}
			if err != hostedexecution.ErrUnavailable || items != nil || calls != wantCalls {
				t.Fatal("incomplete inventory became authoritative")
			}
		})
	}
}

func TestSandboxCancellationDuringCredentialAcquisitionPreventsDispatch(t *testing.T) {
	p := fixturePlan(t, "go", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cred := &credential{t: t, after: cancel}
	client, err := NewClient(p.config, cred, roundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("canceled dispatch reached transport")
		return nil, io.EOF
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Create(ctx, p); err != hostedexecution.ErrUnavailable || cred.calls != 1 {
		t.Fatal("credential acquisition ignored cancellation")
	}
}
