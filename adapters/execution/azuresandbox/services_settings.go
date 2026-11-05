package azuresandbox

import (
	"net/netip"
	"strings"

	"github.com/wtiger001/teststudio/adapters/workerexchange/relay"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ServiceIdentitySettings struct {
	Resource  string `json:"resource"`
	Client    string `json:"client"`
	Principal string `json:"principal"`
}

type RelayServiceSettings struct {
	Application         string                  `json:"application"`
	Certificate         string                  `json:"certificate"`
	Subnet              string                  `json:"subnet"`
	InfrastructureGroup string                  `json:"infrastructureGroup"`
	Image               string                  `json:"image"`
	Pull                ServiceIdentitySettings `json:"pull"`
}

type BackendServiceSettings struct {
	Application         string                  `json:"application"`
	Environment         string                  `json:"environment"`
	Subnet              string                  `json:"subnet"`
	InfrastructureGroup string                  `json:"infrastructureGroup"`
	Origin              string                  `json:"origin"`
	Address             string                  `json:"address"`
	Image               string                  `json:"image"`
	Pull                ServiceIdentitySettings `json:"pull"`
	Startup             ServiceIdentitySettings `json:"startup"`
	Runtime             ServiceIdentitySettings `json:"runtime"`
	DatabaseSecretURL   string                  `json:"databaseSecretUrl"`
	BlobServiceURL      string                  `json:"blobServiceUrl"`
	EvidenceContainer   string                  `json:"evidenceContainer"`
	Replicas            int32                   `json:"replicas"`
}

type ServicesSettings struct {
	RelayNetwork    RelayNetworkSettings    `json:"relayNetwork"`
	Relay           RelayServiceSettings    `json:"relay"`
	Backend         BackendServiceSettings  `json:"backend"`
	BackendEndpoint BackendEndpointSettings `json:"backendEndpoint"`
}

func serviceSubnet(c Config, id string) bool {
	parts := strings.Split(id, "/")
	return len(parts) == 11 && parts[9] == "subnets" && identifier.MatchString(parts[10]) && resourceID(c, strings.Join(parts[:9], "/"), "Microsoft.Network", "virtualNetworks")
}

func trustedImage(c Config, image string) bool {
	base, digest, found := strings.Cut(image, "@sha256:")
	repo := strings.TrimPrefix(base, c.Registry+"/")
	return found && len(image) <= 1024 && strings.HasPrefix(base, c.Registry+"/") && repositoryName.MatchString(repo) && checksum.MatchString(digest) && repo != c.RepositoryPrefix && !strings.HasPrefix(repo, c.RepositoryPrefix+"/")
}

func serviceGroup(value string) bool {
	return groupName.MatchString(value) && !strings.HasSuffix(value, ".") && !strings.Contains(value, "..")
}

func (v ServicesSettings) identities() []ServiceIdentitySettings {
	return []ServiceIdentitySettings{v.Relay.Pull, v.Backend.Pull, v.Backend.Startup, v.Backend.Runtime}
}

func (v ServicesSettings) validate(c Config, p PlacementSettings) error {
	r, b := v.Relay, v.Backend
	certificate := strings.TrimPrefix(r.Certificate, p.Private.ExchangeEnvironment+"/certificates/")
	address, err := netip.ParseAddr(b.Address)
	worker, _ := netip.ParsePrefix(p.Network.Topology.Prefix)
	if !resourceID(c, r.Application, "Microsoft.App", "containerApps") || !resourceID(c, b.Application, "Microsoft.App", "containerApps") || strings.EqualFold(r.Application, b.Application) || !resourceID(c, b.Environment, "Microsoft.App", "managedEnvironments") || strings.EqualFold(b.Environment, p.Private.ExchangeEnvironment) || !identifier.MatchString(certificate) || r.Certificate != p.Private.ExchangeEnvironment+"/certificates/"+certificate || !trustedImage(c, r.Image) || !trustedImage(c, b.Image) || !serviceSubnet(c, r.Subnet) || !serviceSubnet(c, b.Subnet) || !serviceGroup(r.InfrastructureGroup) || !serviceGroup(b.InfrastructureGroup) || strings.EqualFold(r.InfrastructureGroup, b.InfrastructureGroup) || strings.EqualFold(r.InfrastructureGroup, c.ResourceGroup) || strings.EqualFold(b.InfrastructureGroup, c.ResourceGroup) || relay.ValidateOrigins(c.ExchangeOrigin, b.Origin) != nil || err != nil || !address.Is4() || !address.IsPrivate() || address.String() != b.Address || worker.Contains(address) || b.Replicas < 1 || b.Replicas > 10 || !validBackendDependencies(b) {
		return hostedexecution.ErrConfiguration
	}
	networks := map[string]bool{strings.ToLower(p.virtualNetwork()): true}
	for _, subnet := range []string{r.Subnet, b.Subnet} {
		id := strings.ToLower(strings.Join(strings.Split(subnet, "/")[:9], "/"))
		if networks[id] {
			return hostedexecution.ErrConfiguration
		}
		networks[id] = true
	}
	for _, application := range []string{r.Application, b.Application} {
		name := application[strings.LastIndex(application, "/")+1:]
		if len(name) > 32 || strings.Contains(name, "--") {
			return hostedexecution.ErrConfiguration
		}
	}
	resources, clients, principals := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, identity := range v.identities() {
		id := strings.ToLower(identity.Resource)
		if !c.validPullIdentity(identity.Resource) || !nonzeroGUID(identity.Client) || !nonzeroGUID(identity.Principal) || identity.Client == identity.Principal || resources[id] || clients[identity.Client] || principals[identity.Principal] {
			return hostedexecution.ErrConfiguration
		}
		resources[id], clients[identity.Client], principals[identity.Principal] = true, true, true
	}
	if err := v.BackendEndpoint.validate(c, p, v); err != nil {
		return err
	}
	return v.RelayNetwork.validate(c, p, v)
}

func (v ServicesSettings) validateIdentities(s Settings) error {
	resources := []string{s.PullIdentity, s.Timer.StopIdentity, s.Authority.Control.Resource, s.Authority.Registry.Resource, s.Authority.Metadata.Resource}
	clients := []string{s.Placement.PullClientID, s.Authority.StopClientID, s.ControlClientID, s.RegistryClientID, s.MetadataClientID}
	principals := []string{s.Placement.PullPrincipalID, s.Authority.StopPrincipalID, s.Authority.Control.PrincipalID, s.Authority.Registry.PrincipalID, s.Authority.Metadata.PrincipalID}
	for _, identity := range v.identities() {
		for i, id := range resources {
			if strings.EqualFold(identity.Resource, id) || identity.Client == clients[i] || identity.Principal == principals[i] {
				return hostedexecution.ErrConfiguration
			}
		}
	}
	return nil
}
