package azureworker

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

type readDenyFixture struct {
	t           *testing.T
	path        string
	assignments []a.DenyAssignment
	reads       int
	mode        string
}

func readDenyTest(t *testing.T) (Config, ReadIdentitySettings, *readDenyFixture, *ReadDenyGuard) {
	t.Helper()
	c, s, previous, identity := readIdentityTest(t)
	f := &readDenyFixture{t: t, path: observerScope(c) + "/providers/Microsoft.Authorization/denyAssignments"}
	for _, binding := range s.Protection.bindings() {
		f.assignments = append(f.assignments, previous.protection.assignments[binding.assignment])
	}
	reader, err := NewSDKReadDenyReader(c, fixtureCredential{}, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reader.Close)
	g, err := NewReadDenyGuard(c, s, identity, reader)
	if err != nil {
		t.Fatal(err)
	}
	return c, s, f, g
}

func (f *readDenyFixture) extra(scope, action string) a.DenyAssignment {
	assignment := clone(f.assignments[0])
	name := "99999999-aaaa-bbbb-cccc-111111111111"
	assignment.ID = to.Ptr(strings.TrimSuffix(scope, "/") + "/providers/Microsoft.Authorization/denyAssignments/" + name)
	assignment.Name, assignment.Properties.Scope = to.Ptr(name), to.Ptr(scope)
	assignment.Properties.Permissions = []*a.DenyAssignmentPermission{{Actions: []*string{to.Ptr(action)}}}
	return assignment
}

func (f *readDenyFixture) Do(r *http.Request) (*http.Response, error) {
	f.reads++
	if r.Method != "GET" || r.URL.Scheme != "https" || r.URL.Host != "management.azure.com" || r.URL.Path != f.path || r.URL.Query().Get("api-version") != networkDenyVersion || r.Header.Get("Authorization") != "Bearer local-credential-canary" {
		f.t.Error("subscription deny reader widened its endpoint or credential")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	if f.mode == "unavailable" {
		return deadlineResponse(r, 503, map[string]any{"error": map[string]any{"code": "Unavailable", "message": "private-read-deny-canary"}}), nil
	}
	if f.mode == "redirect" {
		response := deadlineResponse(r, 302, nil)
		response.Header.Set("Location", "https://foreign.invalid/private")
		return response, nil
	}
	page := map[string]any{"value": f.assignments}
	switch f.mode {
	case "missing-value":
		delete(page, "value")
	case "null-value":
		page["value"] = nil
	case "null-item":
		page["value"] = []any{nil}
	case "paged":
		if r.URL.Query().Get("$skipToken") == "" {
			page["value"] = f.assignments[:2]
			page["nextLink"] = "https://management.azure.com" + f.path + "?api-version=" + networkDenyVersion + "&%24skipToken=one"
		} else {
			page["value"] = f.assignments[2:]
		}
	case "foreign-page":
		page["nextLink"] = "https://foreign.invalid/private"
	case "wrong-api-page":
		page["nextLink"] = "https://management.azure.com" + f.path + "?api-version=2022-04-01&%24skipToken=one"
	case "endless":
		page["value"] = []any{}
		page["nextLink"] = "https://management.azure.com" + f.path + "?api-version=" + networkDenyVersion + "&%24skipToken=" + strconv.Itoa(f.reads)
	}
	return deadlineResponse(r, 200, page), nil
}
