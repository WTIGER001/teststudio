package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ServiceIdentityPin struct{ Client, Principal, Lifecycle string }

func VerifyServiceIdentityPins(app a.ContainerApp, expected map[string]ServiceIdentityPin) error {
	wanted, pins := map[string]string{}, map[string]serviceIdentityPin{}
	for id, pin := range expected {
		key := inventoryIdentityKey(id)
		wanted[key] = pin.Lifecycle
		pins[key] = serviceIdentityPin{pin.Client, pin.Principal}
	}
	return verifyServiceIdentities(app, wanted, pins)
}

type ServiceApplicationBinding struct {
	Application, Environment, Region, Domain, Certificate string
	Backend                                               bool
	Expected                                              a.ContainerApp
}

func (b ServiceApplicationBinding) Verify(app a.ContainerApp, revisions []a.Revision) error {
	if b.Expected.Identity == nil || b.Expected.Properties == nil || b.Expected.Properties.Template == nil {
		return hostedexecution.ErrConfiguration
	}
	normalize := exchangeTemplateDocument
	if b.Backend {
		normalize = backendTemplateDocument
	}
	identities := []string{}
	for id := range b.Expected.Identity.UserAssignedIdentities {
		identities = append(identities, id)
	}
	binding := serviceBinding{application: b.Application, environment: b.Environment, location: b.Region, domain: b.Domain, certificate: b.Certificate, identities: identities, templateDocument: normalize}
	doc, err := binding.applicationDocument(app)
	if err != nil {
		return err
	}
	if fingerprint(doc) != fingerprint(expectedServiceDocument(b.Expected, normalize)) {
		return hostedexecution.ErrOwnership
	}
	return verifyServiceRevisions(app, revisions, b.Application, b.Expected.Properties.Template, normalize)
}

func VerifyPrivateServiceEnvironment(actual a.ManagedEnvironment, environment, subnet, infrastructure, region string) error {
	return verifyPrivateServiceEnvironment(actual, environment, subnet, infrastructure, region)
}

func VerifyServiceEndpointConnections(actual a.ManagedEnvironment, connections []a.PrivateEndpointConnection, endpoint string) error {
	if actual.Properties == nil || actual.ID == nil {
		return hostedexecution.ErrUnavailable
	}
	id, err := verifyEnvironmentEndpointConnections(connections, *actual.ID, endpoint)
	if err != nil {
		return err
	}
	embedded := []a.PrivateEndpointConnection{}
	for _, connection := range actual.Properties.PrivateEndpointConnections {
		if connection == nil {
			return hostedexecution.ErrOwnership
		}
		embedded = append(embedded, *connection)
	}
	observed, err := verifyEnvironmentEndpointConnections(embedded, *actual.ID, endpoint)
	if err != nil {
		return err
	}
	if id != observed {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func VerifyServicePrivateLinks(links []a.PrivateLinkResource, environment, region string) error {
	_, err := verifyBackendEndpointLink(links, environment, region)
	return err
}

func VerifyServiceAuth(auth a.AuthConfig, application string) error {
	if !strings.EqualFold(value(auth.ID), application+"/authConfigs/current") || value(auth.Name) != "current" || !strings.EqualFold(value(auth.Type), "Microsoft.App/containerApps/authConfigs") {
		return hostedexecution.ErrOwnership
	}
	if auth.Properties == nil || auth.Properties.Platform == nil || auth.Properties.Platform.Enabled == nil {
		return hostedexecution.ErrUnavailable
	}
	if value(auth.Properties.Platform.Enabled) {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func ValidBackendDependencies(secret, blob, container string) bool {
	return backendSecret.MatchString(secret) && backendBlob.MatchString(blob) && backendContainer.MatchString(container) && !strings.Contains(container, "--")
}
