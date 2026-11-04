package azureworker

import (
	"crypto/x509"
	"net/url"
	"strings"
	"time"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (g *ExchangeGuard) verifyEnvironment(actual a.ManagedEnvironment) error {
	return verifyPrivateServiceEnvironment(actual, g.settings.environment(), g.settings.Subnet, g.settings.InfrastructureGroup, g.worker.Region)
}

func verifyPrivateServiceEnvironment(actual a.ManagedEnvironment, environment, subnet, infrastructure, location string) error {
	if err := environmentMatches(actual, environment, subnet, location); err != nil {
		return err
	}
	p := actual.Properties
	if !value(p.VnetConfiguration.Internal) || value(p.PublicNetworkAccess) != a.PublicNetworkAccessDisabled ||
		!strings.EqualFold(value(p.InfrastructureResourceGroup), infrastructure) ||
		p.AppLogsConfiguration == nil ||
		value(p.AppLogsConfiguration.Destination) != "none" ||
		p.AppLogsConfiguration.LogAnalyticsConfiguration != nil ||
		p.DaprAIConnectionString != nil ||
		p.DaprAIInstrumentationKey != nil ||
		len(p.WorkloadProfiles) != 1 ||
		p.WorkloadProfiles[0] == nil ||
		value(p.WorkloadProfiles[0].Name) != "Consumption" ||
		value(p.WorkloadProfiles[0].WorkloadProfileType) != "Consumption" ||
		value(p.WorkloadProfiles[0].MinimumCount) != 0 ||
		value(p.WorkloadProfiles[0].MaximumCount) != 0 {
		return hostedexecution.ErrOwnership
	}
	if actual.Identity != nil && (value(actual.Identity.Type) != a.ManagedServiceIdentityTypeNone || len(actual.Identity.UserAssignedIdentities) != 0) {
		return hostedexecution.ErrOwnership
	}
	if p.CustomDomainConfiguration != nil && (value(p.CustomDomainConfiguration.DNSSuffix) != "" || p.CustomDomainConfiguration.CertificateKeyVaultProperties != nil || len(p.CustomDomainConfiguration.CertificateValue) != 0 || value(p.CustomDomainConfiguration.CertificatePassword) != "") {
		return hostedexecution.ErrOwnership
	}
	domain := value(p.DefaultDomain)
	if domain == "" {
		return hostedexecution.ErrUnavailable
	}
	if !hostname.MatchString(domain) || strings.Contains(domain, "..") || !strings.HasSuffix(domain, "."+location+".azurecontainerapps.io") {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (g *ExchangeGuard) verifyExchangeConnections(connections []a.PrivateEndpointConnection) (string, error) {
	if len(connections) == 0 {
		return "", hostedexecution.ErrUnavailable
	}
	if len(connections) != 1 {
		return "", hostedexecution.ErrOwnership
	}
	connection := connections[0]
	if !endpointChild(value(connection.ID), value(connection.Name), g.settings.environment(), "privateEndpointConnections") ||
		!strings.EqualFold(value(connection.Type), "Microsoft.App/managedEnvironments/privateEndpointConnections") ||
		connection.Properties == nil ||
		connection.Properties.PrivateEndpoint == nil ||
		!strings.EqualFold(value(connection.Properties.PrivateEndpoint.ID), g.settings.Registry.Endpoints.ExchangeEndpoint) {
		return "", hostedexecution.ErrOwnership
	}
	p := connection.Properties
	if len(p.GroupIDs) > 0 && !registryStrings(p.GroupIDs, "managedEnvironment") {
		return "", hostedexecution.ErrOwnership
	}
	if value(p.ProvisioningState) != a.PrivateEndpointConnectionProvisioningStateSucceeded || p.PrivateLinkServiceConnectionState == nil {
		return "", hostedexecution.ErrUnavailable
	}
	state := p.PrivateLinkServiceConnectionState
	switch value(state.Status) {
	case a.PrivateEndpointServiceConnectionStatusApproved:
	case "", a.PrivateEndpointServiceConnectionStatusPending:
		return "", hostedexecution.ErrUnavailable
	default:
		return "", hostedexecution.ErrOwnership
	}
	if action := value(state.ActionsRequired); action != "" && action != "None" {
		return "", hostedexecution.ErrUnavailable
	}
	return strings.ToLower(value(connection.ID)), nil
}

func (g *ExchangeGuard) verifyExchangeLinks(links []a.PrivateLinkResource) error {
	if len(links) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if len(links) != 1 {
		return hostedexecution.ErrOwnership
	}
	link := links[0]
	if !endpointChild(value(link.ID), value(link.Name), g.settings.environment(), "privateLinkResources") || !strings.EqualFold(value(link.Type), "Microsoft.App/managedEnvironments/privateLinkResources") || link.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := link.Properties
	if p.GroupID == nil || len(p.RequiredMembers) == 0 || len(p.RequiredZoneNames) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if value(p.GroupID) != "managedEnvironment" || len(p.RequiredMembers) != 1 || !endpointChildName.MatchString(value(p.RequiredMembers[0])) || !registryStrings(p.RequiredZoneNames, "privatelink."+g.worker.Region+".azurecontainerapps.io") {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (g *ExchangeGuard) verifyCertificate(actual a.Certificate, now time.Time) error {
	if !dnsIdentity(value(actual.ID), value(actual.Name), value(actual.Type), g.settings.Certificate, "Microsoft.App/managedEnvironments/certificates", value(actual.Location), g.worker.Region) || actual.Properties == nil {
		return hostedexecution.ErrOwnership
	}
	p := actual.Properties
	if value(p.ProvisioningState) != a.CertificateProvisioningStateSucceeded || p.Valid == nil || !value(p.Valid) || p.IssueDate == nil || p.ExpirationDate == nil || p.IssueDate.After(now) || !p.ExpirationDate.After(now.Add(2*time.Hour+5*time.Minute)) {
		return hostedexecution.ErrUnavailable
	}
	if p.CertificateKeyVaultProperties != nil || value(p.Password) != "" || len(p.Value) != 0 || len(p.SubjectAlternativeNames) > 64 {
		return hostedexecution.ErrOwnership
	}
	names := []string{}
	for _, entry := range p.SubjectAlternativeNames {
		if entry == nil {
			return hostedexecution.ErrOwnership
		}
		names = append(names, *entry)
	}
	origin, _ := url.Parse(g.worker.ExchangeOrigin)
	certificate := x509.Certificate{DNSNames: names}
	if certificate.VerifyHostname(origin.Hostname()) != nil {
		return hostedexecution.ErrOwnership
	}
	return nil
}
