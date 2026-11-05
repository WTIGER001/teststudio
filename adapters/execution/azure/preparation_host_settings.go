package azureworker

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

const PreparationJournalDirectory = "/private/source-journals"
const PreparationJournalMountOptions = "dir_mode=0700,file_mode=0600,uid=65532,gid=65532,cache=none"

var preparationAccount = regexp.MustCompile(`^[a-z0-9]{3,24}$`)

type PreparationHostSettings struct {
	Application            string `json:"application"`
	Environment            string `json:"environment"`
	Image                  string `json:"image"`
	ReaderIdentity         string `json:"readerIdentity"`
	ReaderPrincipalID      string `json:"readerPrincipalId"`
	ReaderAssignment       string `json:"readerAssignment"`
	ReaderRole             string `json:"readerRole"`
	PullIdentity           string `json:"pullIdentity"`
	StartupIdentity        string `json:"startupIdentity"`
	JournalStorageName     string `json:"journalStorageName"`
	JournalAccount         string `json:"journalAccount"`
	JournalShare           string `json:"journalShare"`
	DatabaseSecretURL      string `json:"databaseSecretUrl"`
	ConfigurationSecretURL string `json:"configurationSecretUrl"`
	GoRuntime              string `json:"goRuntime,omitempty"`
	PlaywrightRuntime      string `json:"playwrightRuntime,omitempty"`
}

func (h PreparationHostSettings) canonical(c Config, w PreparationWriterSettings, reader string) (PreparationHostSettings, error) {
	if !c.ownsResource(h.Application, "Microsoft.App", "containerApps") || !c.ownsResource(h.Environment, "Microsoft.App", "managedEnvironments") || !trustedServiceImage(c, h.Image) || !resourceName.MatchString(h.JournalStorageName) || !preparationAccount.MatchString(h.JournalAccount) || !backendContainer.MatchString(h.JournalShare) || strings.Contains(h.JournalShare, "--") || !observerGUID(reader) || reader == w.ClientID || !observerGUID(h.ReaderPrincipalID) || h.ReaderPrincipalID == reader || h.ReaderPrincipalID == w.PrincipalID || !observerAuthorizationID(c, h.ReaderAssignment, "roleAssignments") || !observerAuthorizationID(c, h.ReaderRole, "roleDefinitions") {
		return h, hostedexecution.ErrConfiguration
	}
	for _, id := range []string{c.WorkerEnvironment, c.ControllerEnvironment, w.Registry.Registry.Endpoints.ExchangeEnvironment} {
		if strings.EqualFold(h.Environment, id) {
			return h, hostedexecution.ErrConfiguration
		}
	}
	seen := map[string]bool{inventoryIdentityKey(c.PullIdentity): true}
	for _, id := range h.identities(w) {
		key := inventoryIdentityKey(id)
		if !c.ownsResource(id, "Microsoft.ManagedIdentity", "userAssignedIdentities") || seen[key] {
			return h, hostedexecution.ErrConfiguration
		}
		seen[key] = true
	}
	if !backendSecret.MatchString(h.DatabaseSecretURL) || !backendSecret.MatchString(h.ConfigurationSecretURL) || strings.EqualFold(h.DatabaseSecretURL, h.ConfigurationSecretURL) {
		return h, hostedexecution.ErrConfiguration
	}
	database, _ := url.Parse(h.DatabaseSecretURL)
	configuration, _ := url.Parse(h.ConfigurationSecretURL)
	if database.Host != configuration.Host {
		return h, hostedexecution.ErrConfiguration
	}
	registry, err := oci.New(oci.Options{Origin: "https://" + c.Registry, RepositoryPrefix: c.RepositoryPrefix, Architecture: "amd64"})
	if err != nil {
		return h, hostedexecution.ErrConfiguration
	}
	defer registry.Close()
	for _, image := range []string{h.GoRuntime, h.PlaywrightRuntime} {
		if image != "" && registry.ValidateReference(image) != nil {
			return h, hostedexecution.ErrConfiguration
		}
	}
	return h, nil
}

func (h PreparationHostSettings) identities(w PreparationWriterSettings) []string {
	return []string{w.Identity, h.ReaderIdentity, h.PullIdentity, h.StartupIdentity}
}

func (h PreparationHostSettings) reader(s PreparationSettings) readIdentityBinding {
	return readIdentityBinding{Identity: h.ReaderIdentity, ClientID: s.ReaderClientID, PrincipalID: h.ReaderPrincipalID, TenantID: s.Writer.TenantID, Assignment: h.ReaderAssignment, Role: h.ReaderRole}
}

func (h PreparationHostSettings) storage() string {
	return h.Environment + "/storages/" + h.JournalStorageName
}
