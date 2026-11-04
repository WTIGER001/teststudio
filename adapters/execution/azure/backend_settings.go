package azureworker

import (
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type BackendSettings struct {
	Exchange            ExchangeSettings `json:"exchange"`
	Application         string           `json:"application"`
	Environment         string           `json:"environment"`
	Subnet              string           `json:"subnet"`
	InfrastructureGroup string           `json:"infrastructureGroup"`
	Image               string           `json:"image"`
	PullIdentity        string           `json:"pullIdentity"`
	StartupIdentity     string           `json:"startupIdentity"`
	RuntimeIdentity     string           `json:"runtimeIdentity"`
	RuntimeClientID     string           `json:"runtimeClientId"`
	DatabaseSecretURL   string           `json:"databaseSecretUrl"`
	BlobServiceURL      string           `json:"blobServiceUrl"`
	EvidenceContainer   string           `json:"evidenceContainer"`
	Replicas            int32            `json:"replicas"`
}

var backendSecret = regexp.MustCompile(`^https://[a-z][a-z0-9-]{1,22}[a-z0-9]\.vault\.azure\.net/secrets/[A-Za-z0-9-]{1,127}$`)
var backendBlob = regexp.MustCompile(`^https://[a-z0-9]{3,24}\.blob\.core\.windows\.net/$`)
var backendContainer = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

func (s BackendSettings) canonical(c Config) (BackendSettings, error) {
	var err error
	s.Exchange, err = s.Exchange.canonical(c)
	if err != nil {
		return s, hostedexecution.ErrConfiguration
	}
	topology := s.Exchange.Registry.Endpoints.PrivateDNS.DNS.Firewall.Perimeter.Topology
	if s.Replicas == 0 {
		s.Replicas = 1
	}
	originConfig := c
	originConfig.ExchangeOrigin = s.Exchange.BackendOrigin
	if Validate(originConfig) != nil || !c.ownsResource(s.Application, "Microsoft.App", "containerApps") ||
		strings.EqualFold(s.Application, s.Exchange.Application) || !c.ownsResource(s.Environment, "Microsoft.App", "managedEnvironments") ||
		strings.EqualFold(s.Environment, c.WorkerEnvironment) || strings.EqualFold(s.Environment, c.ControllerEnvironment) || strings.EqualFold(s.Environment, s.Exchange.environment()) ||
		!subnetResource(c, s.Subnet, topology.ControllerVNet) || strings.EqualFold(s.Subnet, topology.ControllerSubnet) ||
		!groupName.MatchString(s.InfrastructureGroup) || strings.HasSuffix(s.InfrastructureGroup, ".") ||
		strings.EqualFold(s.InfrastructureGroup, c.ResourceGroup) || strings.EqualFold(s.InfrastructureGroup, topology.InfrastructureGroup) || strings.EqualFold(s.InfrastructureGroup, s.Exchange.InfrastructureGroup) ||
		!guid.MatchString(s.RuntimeClientID) || !backendSecret.MatchString(s.DatabaseSecretURL) || !backendBlob.MatchString(s.BlobServiceURL) ||
		!backendContainer.MatchString(s.EvidenceContainer) || strings.Contains(s.EvidenceContainer, "--") || s.Replicas < 1 || s.Replicas > 10 || !trustedServiceImage(c, s.Image) {
		return s, hostedexecution.ErrConfiguration
	}
	_, name := resourceParts(s.Application)
	if len(name) > 32 || strings.Contains(name, "--") {
		return s, hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{strings.ToLower(c.PullIdentity): true}
	for _, id := range []string{s.PullIdentity, s.StartupIdentity, s.RuntimeIdentity} {
		if !c.ownsResource(id, "Microsoft.ManagedIdentity", "userAssignedIdentities") || seen[strings.ToLower(id)] {
			return s, hostedexecution.ErrConfiguration
		}
		seen[strings.ToLower(id)] = true
	}
	return s, nil
}

func trustedServiceImage(c Config, image string) bool {
	base, digest, ok := strings.Cut(image, "@sha256:")
	repo := strings.TrimPrefix(base, c.Registry+"/")
	return ok && strings.HasPrefix(base, c.Registry+"/") && repository.MatchString(repo) && checksum.MatchString(digest) && len(image) <= 1024 && repo != c.RepositoryPrefix && !strings.HasPrefix(repo, c.RepositoryPrefix+"/")
}
