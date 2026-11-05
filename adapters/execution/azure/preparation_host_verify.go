package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyPreparationHostPlacement(s PreparationSettings, environment a.ManagedEnvironment, storage a.ManagedEnvironmentStorage) error {
	c, h := s.Worker, *s.Host
	if err := environmentMatches(environment, h.Environment, s.Writer.Registry.SourceSubnet, c.Region); err != nil {
		return err
	}
	p := environment.Properties
	if p.PublicNetworkAccess == nil || p.VnetConfiguration.Internal == nil || len(p.WorkloadProfiles) == 0 {
		return hostedexecution.ErrUnavailable
	}
	if value(p.PublicNetworkAccess) != a.PublicNetworkAccessDisabled || !value(p.VnetConfiguration.Internal) || len(p.WorkloadProfiles) != 1 || p.WorkloadProfiles[0] == nil || value(p.WorkloadProfiles[0].Name) != "Consumption" || value(p.WorkloadProfiles[0].WorkloadProfileType) != "Consumption" {
		return hostedexecution.ErrOwnership
	}
	if !endpointChild(value(storage.ID), h.JournalStorageName, h.Environment, "storages") || !strings.EqualFold(value(storage.Name), h.JournalStorageName) || !strings.EqualFold(value(storage.Type), "Microsoft.App/managedEnvironments/storages") {
		return hostedexecution.ErrOwnership
	}
	if storage.Properties != nil && storage.Properties.NfsAzureFile != nil {
		return hostedexecution.ErrOwnership
	}
	if storage.Properties == nil || storage.Properties.AzureFile == nil || storage.Properties.AzureFile.AccountName == nil || storage.Properties.AzureFile.ShareName == nil || storage.Properties.AzureFile.AccessMode == nil {
		return hostedexecution.ErrUnavailable
	}
	file := storage.Properties.AzureFile
	if value(file.AccountName) != h.JournalAccount || value(file.ShareName) != h.JournalShare || value(file.AccessMode) != a.AccessModeReadWrite {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func verifyPreparationHostApplication(s PreparationSettings, app a.ContainerApp, revisions []a.Revision) error {
	c, h, w := s.Worker, *s.Host, s.Writer
	wanted := map[string]string{inventoryIdentityKey(w.Identity): "Main", inventoryIdentityKey(h.ReaderIdentity): "Main", inventoryIdentityKey(h.PullIdentity): "None", inventoryIdentityKey(h.StartupIdentity): "None"}
	pins := map[string]serviceIdentityPin{inventoryIdentityKey(w.Identity): {client: w.ClientID, principal: w.PrincipalID}, inventoryIdentityKey(h.ReaderIdentity): {client: s.ReaderClientID, principal: h.ReaderPrincipalID}}
	if err := verifyServiceIdentities(app, wanted, pins); err != nil {
		return err
	}
	expected, err := s.PreparationApplicationDefinition()
	if err != nil {
		return err
	}
	binding := serviceBinding{application: h.Application, environment: h.Environment, location: c.Region, identities: h.identities(w), templateDocument: backendTemplateDocument, headless: true}
	doc, err := binding.applicationDocument(app)
	if err != nil {
		return err
	}
	properties, _ := doc["properties"].(map[string]any)
	configuration, _ := properties["configuration"].(map[string]any)
	if maximum := app.Properties.Configuration.MaxInactiveRevisions; maximum != nil {
		if *maximum < 1 || *maximum > 100 {
			return hostedexecution.ErrOwnership
		}
		delete(configuration, "maxInactiveRevisions")
	}
	if fingerprint(doc) != fingerprint(expectedServiceDocument(expected, backendTemplateDocument)) {
		return hostedexecution.ErrOwnership
	}
	return verifyServiceRevisions(app, revisions, h.Application, expected.Properties.Template, backendTemplateDocument)
}
