package azureworker

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzurePreparationWriterOpensPinnedPublisherAfterFreshPathAndGrant(t *testing.T) {
	c, s, f, parent, g := preparationWriterTest(t)
	var clients []string
	g.credential = func(id string) (azcore.TokenCredential, error) {
		clients = append(clients, id)
		return fixtureCredential{}, nil
	}
	s.ClientID = "aaaaaaaa-1111-2222-3333-444444444444"
	for i := 1; i <= 2; i++ {
		if i == 2 {
			f.role.ID = to.Ptr(preparationPushRole)
			f.assignment.Properties.RoleDefinitionID = to.Ptr(preparationPushRole)
			slices.Reverse(f.role.Properties.Permissions[0].Actions)
		}
		client, err := g.OpenRegistry(t.Context(), c)
		if err != nil || client == nil {
			t.Fatal("selected writer publisher rejected", err)
		}
		image := c.Registry + "/" + c.RepositoryPrefix + "/runtime@sha256:" + strings.Repeat("a", 64)
		if client.ValidateReference(image) != nil || client.ValidateReference(strings.Replace(image, c.RepositoryPrefix, "foreign", 1)) == nil {
			t.Fatal("publisher repository boundary changed")
		}
		client.Close()
		client.Close()
		if f.reads != 3*i || parent.reads != 6*i || len(clients) != i || clients[i-1] != f.settings.ClientID {
			t.Fatal("writer reused an observation or selected another client", f.reads, parent.reads, clients)
		}
	}
	if _, ok := any(g).(Network); ok {
		t.Fatal("selected publisher grant became complete execution admission")
	}
	f.assignment.Properties.PrincipalID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
	client, err := g.OpenRegistry(t.Context(), c)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if client != nil || len(clients) != 2 {
		t.Fatal("revoked writer reached the credential factory")
	}
	parent.mode = "unavailable"
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	if f.reads != 9 {
		t.Fatal("unverified registry path reached writer observations")
	}
}

func TestAzurePreparationWriterRejectsChangedGrantsAndIncompleteResponses(t *testing.T) {
	for name, mutate := range map[string]func(*preparationWriterFixture){
		"foreign-client": func(f *preparationWriterFixture) {
			f.identity.Properties.ClientID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
		},
		"foreign-tenant": func(f *preparationWriterFixture) {
			f.identity.Properties.TenantID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
		},
		"broad-scope": func(f *preparationWriterFixture) {
			f.assignment.Properties.Scope = to.Ptr(strings.Split(f.settings.Assignment, "/resourceGroups/")[0])
		},
		"condition": func(f *preparationWriterFixture) {
			f.assignment.Properties.Condition = to.Ptr("private-condition-canary")
		},
		"delegation": func(f *preparationWriterFixture) {
			f.assignment.Properties.DelegatedManagedIdentityResourceID = to.Ptr(f.settings.Identity)
		},
		"group": func(f *preparationWriterFixture) {
			f.assignment.Properties.PrincipalType = to.Ptr(a.PrincipalTypeGroup)
		},
		"other-role": func(f *preparationWriterFixture) {
			f.assignment.Properties.RoleDefinitionID = to.Ptr(strings.Replace(preparationPushRole, "8311e382", "8311e383", 1))
		},
		"custom-role": func(f *preparationWriterFixture) { f.role.Properties.RoleType = to.Ptr("CustomRole") },
		"delete-permission": func(f *preparationWriterFixture) {
			f.role.Properties.Permissions[0].Actions = append(f.role.Properties.Permissions[0].Actions, to.Ptr("Microsoft.ContainerRegistry/registries/artifacts/delete"))
		},
		"data-permission": func(f *preparationWriterFixture) {
			f.role.Properties.Permissions[0].DataActions = []*string{to.Ptr("*")}
		},
		"exclusion": func(f *preparationWriterFixture) {
			f.role.Properties.Permissions[0].NotActions = []*string{to.Ptr("*")}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, f, _, g := preparationWriterTest(t)
			mutate(f)
			requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrOwnership)
		})
	}
	c, _, f, _, g := preparationWriterTest(t)
	f.identity.Properties.TenantID = nil
	requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	f.identity.Properties.TenantID = to.Ptr(f.settings.TenantID)
	for _, mode := range []string{"unavailable", "redirect"} {
		f.mode = mode
		requireDeadlineError(t, g.Check(t.Context(), c), hostedexecution.ErrUnavailable)
	}
	f.mode = ""
	g.credential = func(string) (azcore.TokenCredential, error) { return nil, errors.New("private-writer-canary") }
	_, err := g.OpenRegistry(t.Context(), c)
	requireDeadlineError(t, err, hostedexecution.ErrUnavailable)
}

func TestAzurePreparationWriterSettingsAndSDKRejectUnselectedAuthority(t *testing.T) {
	c, s, f, _, g := preparationWriterTest(t)
	data, _ := json.Marshal(PreparationSettings{Worker: c, Writer: s})
	loaded, err := LoadPreparationSettings("", "TESTSTUDIO_AZURE_PREPARATION", func(string) (string, bool) { return string(data), true })
	if err != nil || loaded == nil || fingerprint(loaded.Worker) != fingerprint(c) {
		t.Fatal("explicit preparation settings rejected", err)
	}
	_, err = LoadPreparationSettings("", "TESTSTUDIO_AZURE_PREPARATION", func(string) (string, bool) { return `{"unexpected":"private-writer-canary"}`, true })
	if err == nil || strings.Contains(err.Error(), "private-writer-canary") {
		t.Fatal("unknown preparation setting was accepted or leaked", err)
	}
	for _, mutate := range []func(*PreparationWriterSettings){
		func(s *PreparationWriterSettings) { s.Identity = c.PullIdentity },
		func(s *PreparationWriterSettings) { s.ClientID = s.PrincipalID },
		func(s *PreparationWriterSettings) { s.TenantID = allDenyPrincipals },
		func(s *PreparationWriterSettings) {
			s.Assignment = observerScope(c) + "/providers/Microsoft.Authorization/roleAssignments/44444444-dddd-4444-dddd-444444444444"
		},
	} {
		changed := s
		mutate(&changed)
		_, err := changed.canonical(c)
		requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
	}
	reader := g.reader.(*SDKPreparationWriterReader)
	for _, mutate := range []func(*http.Request){
		func(r *http.Request) { r.Method = "PUT" },
		func(r *http.Request) { r.URL.Path += "/listKeys" },
		func(r *http.Request) { r.URL.Host = "foreign.invalid" },
		func(r *http.Request) { r.URL.RawQuery += "&$filter=principalId" },
	} {
		r, _ := http.NewRequestWithContext(t.Context(), "GET", "https://management.azure.com"+s.Identity+"?api-version="+readIdentityVersion, nil)
		mutate(r)
		_, err := reader.http.Do(r)
		requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	}
	_, err = reader.Identity(t.Context(), c.PullIdentity)
	requireDeadlineError(t, err, hostedexecution.ErrOwnership)
	if f.reads != 0 {
		t.Fatal("unselected writer operation reached transport")
	}
	changed := s
	changed.Registry.RegistryIP = "10.21.3.9"
	_, err = NewPreparationWriterGuard(c, changed, g.registry, reader)
	requireDeadlineError(t, err, hostedexecution.ErrConfiguration)
}
