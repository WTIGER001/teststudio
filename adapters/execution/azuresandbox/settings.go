package azuresandbox

import (
	"strings"

	"github.com/wtiger001/teststudio/adapters/registry/acr"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/runtimeconfig"
)

type TimerSettings struct {
	ResourceGroup string `json:"resourceGroup"`
	NamePrefix    string `json:"namePrefix"`
	StopIdentity  string `json:"stopIdentity"`
}

type Settings struct {
	Worker              Config            `json:"worker"`
	Mode                string            `json:"mode"`
	TenantID            string            `json:"tenantId"`
	ControlClientID     string            `json:"controlClientId"`
	RegistryClientID    string            `json:"registryClientId"`
	MetadataClientID    string            `json:"metadataClientId"`
	Placement           PlacementSettings `json:"placement"`
	PullIdentity        string            `json:"pullIdentity"`
	BlobRedirectOrigins []string          `json:"blobRedirectOrigins,omitempty"`
	Timer               TimerSettings     `json:"timer"`
}

func (s Settings) deadlineConfig() DeadlineConfig {
	return DeadlineConfig{Worker: s.Worker, ResourceGroup: s.Timer.ResourceGroup, NamePrefix: s.Timer.NamePrefix, StopIdentity: s.Timer.StopIdentity}
}

func nonzeroGUID(value string) bool {
	return guid.MatchString(value) && value != "00000000-0000-0000-0000-000000000000"
}

func (s Settings) Validate() error {
	profile, err := s.Placement.NetworkProfile(s.Worker)
	if err != nil || profile != s.Worker.NetworkProfile || Validate(s.Worker) != nil || s.Mode != "execution" && s.Mode != "discovery" || !nonzeroGUID(s.Worker.Subscription) || !nonzeroGUID(s.TenantID) || !nonzeroGUID(s.ControlClientID) || !nonzeroGUID(s.RegistryClientID) || !nonzeroGUID(s.MetadataClientID) || s.ControlClientID == s.RegistryClientID || s.ControlClientID == s.MetadataClientID || s.RegistryClientID == s.MetadataClientID || s.Placement.PullClientID == s.ControlClientID || s.Placement.PullClientID == s.RegistryClientID || s.Placement.PullClientID == s.MetadataClientID || !s.Worker.validPullIdentity(s.PullIdentity) || strings.EqualFold(s.PullIdentity, s.Timer.StopIdentity) || s.deadlineConfig().validate() != nil || acr.ValidateRegistry(s.Worker.Registry, s.TenantID, s.Worker.RepositoryPrefix, s.BlobRedirectOrigins) != nil {
		return hostedexecution.ErrConfiguration
	}
	return nil
}

func LoadSettings(file, variable string, lookup func(string) (string, bool)) (*Settings, error) {
	if variable != "" && lookup == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	var s Settings
	found, err := runtimeconfig.LoadJSON(file, variable, "Azure sandbox", lookup, &s)
	if err != nil || !found {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s Settings) String() string   { return "Azure sandbox runtime settings" }
func (s Settings) GoString() string { return s.String() }
