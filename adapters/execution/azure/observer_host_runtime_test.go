package azureworker

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestAzureObserverHostRuntimeRejectsStartupConfigurationDrift(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	changes := map[string]func(*a.ContainerApp){
		"command": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Command = []*string{to.Ptr("/bin/sh")}
		},
		"role": func(app *a.ContainerApp) { app.Properties.Template.Containers[0].Args[0] = to.Ptr("--serve-role=all") },
		"postgres": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Args[1] = to.Ptr("--require-postgres=false")
		},
		"extra-flag": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Args = append(app.Properties.Template.Containers[0].Args, to.Ptr("--local=true"))
		},
		"public-origin": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Args[2] = to.Ptr("--public-url=https://foreign.invalid")
		},
		"inline-database": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Env[0].Value = to.Ptr("private-db-canary")
		},
		"database-ref": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Env[0].SecretRef = to.Ptr("oidc-configuration")
		},
		"extra-environment": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Env = append(app.Properties.Template.Containers[0].Env, &a.EnvironmentVar{Name: to.Ptr("HTTPS_PROXY"), Value: to.Ptr("http://foreign.invalid")})
		},
		"duplicate-environment": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Env = append(app.Properties.Template.Containers[0].Env, clone(app.Properties.Template.Containers[0].Env[0]))
		},
		"storage-driver": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Env[1].Value = to.Ptr(`{"driver":"filesystem"}`)
		},
		"storage-client": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Env[1].Value = to.Ptr(strings.ReplaceAll(value(app.Properties.Template.Containers[0].Env[1].Value), s.Runtime.RuntimeClientID, s.Identity.ClientID))
		},
		"secret-url": func(app *a.ContainerApp) {
			app.Properties.Configuration.Secrets[0].KeyVaultURL = to.Ptr(s.Runtime.OIDCConfigurationSecretURL)
		},
		"secret-identity": func(app *a.ContainerApp) {
			app.Properties.Configuration.Secrets[0].Identity = to.Ptr(s.Identity.Identity)
		},
		"inline-secret": func(app *a.ContainerApp) {
			app.Properties.Configuration.Secrets[0].Value = to.Ptr("private-secret-canary")
		},
		"pull-identity": func(app *a.ContainerApp) {
			app.Properties.Configuration.Registries[0].Identity = to.Ptr(s.RuntimeIdentity)
		},
		"registry-password": func(app *a.ContainerApp) {
			app.Properties.Configuration.Registries[0].PasswordSecretRef = to.Ptr("database")
		},
		"insecure-ingress": func(app *a.ContainerApp) { app.Properties.Configuration.Ingress.AllowInsecure = to.Ptr(true) },
		"wrong-port":       func(app *a.ContainerApp) { app.Properties.Configuration.Ingress.TargetPort = to.Ptr[int32](8042) },
		"wrong-host":       func(app *a.ContainerApp) { app.Properties.Configuration.Ingress.Fqdn = to.Ptr("foreign.invalid") },
		"traffic":          func(app *a.ContainerApp) { app.Properties.Configuration.Ingress.Traffic[0].Weight = to.Ptr[int32](50) },
		"volume":           func(app *a.ContainerApp) { app.Properties.Template.Volumes = []*a.Volume{{Name: to.Ptr("foreign")}} },
		"volume-mount": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].VolumeMounts = []*a.VolumeMount{{VolumeName: to.Ptr("foreign"), MountPath: to.Ptr("/app")}}
		},
		"capacity": func(app *a.ContainerApp) { app.Properties.Template.Containers[0].Resources.CPU = to.Ptr(1.0) },
		"replicas": func(app *a.ContainerApp) { app.Properties.Template.Scale.MinReplicas = to.Ptr[int32](0) },
		"probe": func(app *a.ContainerApp) {
			app.Properties.Template.Containers[0].Probes[0].HTTPGet.Path = to.Ptr("/wrong")
		},
		"runtime-metadata": func(app *a.ContainerApp) {
			app.Identity.UserAssignedIdentities[s.RuntimeIdentity].ClientID = to.Ptr("aaaaaaaa-1111-2222-3333-444444444444")
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			app, revisions := clone(f.application), clone(f.revisions)
			change(&app)
			revisions[0].Properties.Template = clone(app.Properties.Template)
			requireReadDenyError(t, verifyObserverHost(c, s, app, revisions), hostedexecution.ErrOwnership)
		})
	}
	app := clone(f.application)
	app.Properties.Configuration.Ingress = nil
	requireReadDenyError(t, verifyObserverHost(c, s, app, f.revisions), hostedexecution.ErrUnavailable)
}

func TestAzureObserverHostRuntimeNormalizesPresentationAndPinsPackagedArguments(t *testing.T) {
	c, s, f, _ := observerHostTest(t)
	expected := []string{"--serve-role=api", "--require-postgres=true", "--public-url=" + s.Runtime.UIOrigin, "--artifact-origin=" + s.Runtime.ArtifactOrigin, "--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION", "--execution-location=azure-container-apps", "--oidc-config-env=TESTSTUDIO_OIDC_CONFIGURATION", "--oidc-secret-refs=env:TESTSTUDIO_OIDC_CLIENT_ONE,env:TESTSTUDIO_OIDC_CLIENT_TWO", "--bootstrap-issuer=https://login.example.test/tenant", "--bootstrap-subject=teststudio-owner", "--local-accounts=false"}
	container := f.application.Properties.Template.Containers[0]
	if len(container.Command) != 0 || len(container.Args) != len(expected) {
		t.Fatal("host changed the packaged image entrypoint or API arguments")
	}
	for i, arg := range expected {
		if value(container.Args[i]) != arg {
			t.Fatal("unexpected packaged API argument", i)
		}
	}
	var storage map[string]any
	if json.Unmarshal([]byte(value(container.Env[1].Value)), &storage) != nil {
		t.Fatal("invalid storage configuration")
	}
	pretty, _ := json.MarshalIndent(storage, "", "  ")
	container.Env[1].Value = to.Ptr(string(pretty))
	slices.Reverse(container.Env)
	slices.Reverse(container.Probes)
	for _, probe := range container.Probes {
		probe.SuccessThreshold = to.Ptr[int32](1)
	}
	slices.Reverse(f.application.Properties.Configuration.Secrets)
	slices.Reverse(f.application.Properties.Configuration.IdentitySettings)
	f.application.Properties.WorkloadProfileName = to.Ptr("Consumption")
	f.application.Properties.Configuration.MaxInactiveRevisions = to.Ptr[int32](100)
	f.application.Properties.Template.Scale.CooldownPeriod = to.Ptr[int32](300)
	f.application.Properties.Template.Scale.PollingInterval = to.Ptr[int32](30)
	if err := verifyObserverHost(c, s, f.application, f.revisions); err != nil {
		t.Fatal("equivalent packaged runtime was rejected", err)
	}
	withoutClients := s
	withoutClients.Runtime.OIDCClientOneSecretURL, withoutClients.Runtime.OIDCClientTwoSecretURL = "", ""
	app, err := withoutClients.ApplicationDefinition(c)
	if err != nil || len(app.Properties.Configuration.Secrets) != 2 || len(app.Properties.Template.Containers[0].Env) != 3 || value(app.Properties.Template.Containers[0].Args[7]) != "--oidc-secret-refs=" {
		t.Fatal("public OIDC clients required a client secret", err)
	}
}

func TestAzureObserverHostRuntimeSettingsRejectUnboundInputsBeforeBootstrap(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	for name, change := range map[string]func(*ObserverHostRuntime){
		"missing":         func(r *ObserverHostRuntime) { *r = ObserverHostRuntime{} },
		"observer-client": func(r *ObserverHostRuntime) { r.RuntimeClientID = s.Identity.ClientID },
		"database-inline": func(r *ObserverHostRuntime) { r.DatabaseSecretURL = "postgres://private-canary" },
		"same-secret":     func(r *ObserverHostRuntime) { r.DatabaseSecretURL = r.OIDCConfigurationSecretURL },
		"other-vault": func(r *ObserverHostRuntime) {
			r.OIDCClientOneSecretURL = "https://othervault.vault.azure.net/secrets/client"
		},
		"storage-url":     func(r *ObserverHostRuntime) { r.BlobServiceURL += "?sig=private-canary" },
		"replicas":        func(r *ObserverHostRuntime) { r.Replicas = 11 },
		"same-origin":     func(r *ObserverHostRuntime) { r.ArtifactOrigin = r.UIOrigin },
		"origin-port":     func(r *ObserverHostRuntime) { r.UIOrigin += ":443" },
		"other-api":       func(r *ObserverHostRuntime) { r.UIOrigin = "https://other.controllerzone.eastus.azurecontainerapps.io" },
		"other-domain":    func(r *ObserverHostRuntime) { r.ArtifactOrigin = "https://artifacts.foreign.invalid" },
		"issuer-query":    func(r *ObserverHostRuntime) { r.BootstrapIssuer += "?secret=private-canary" },
		"issuer-user":     func(r *ObserverHostRuntime) { r.BootstrapIssuer = "https://private-canary@login.example.test" },
		"subject-control": func(r *ObserverHostRuntime) { r.BootstrapSubject += "\nprivate-canary" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := s
			change(&changed.Runtime)
			_, err := g.BindHostObserver(t.Context(), c, changed, f)
			requireReadDenyError(t, err, hostedexecution.ErrConfiguration)
		})
	}
	if f.parent.bootstrapReads != 0 || len(f.parent.factoryClients) != 0 {
		t.Fatal("invalid runtime settings reached bootstrap or credential construction")
	}
}

func TestAzureObserverHostRuntimeRefreshesConsistentAppAndRevisionDrift(t *testing.T) {
	c, s, f, g := observerHostTest(t)
	observer, err := g.BindHostObserver(t.Context(), c, s, f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	bootstrap := f.parent.bootstrapReads
	if err := observer.Check(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	original := clone(f.application.Properties.Template)
	f.application.Properties.Template.Containers[0].Args = append(f.application.Properties.Template.Containers[0].Args, to.Ptr("--local=true"))
	f.revisions[0].Properties.Template = clone(f.application.Properties.Template)
	requireReadDenyError(t, observer.Check(t.Context(), c), hostedexecution.ErrOwnership)
	f.application.Properties.Template = original
	f.revisions[0].Properties.Template = clone(original)
	if err := observer.Check(t.Context(), c); err != nil {
		t.Fatal("restored runtime rejected", err)
	}
	if f.reads != 6 || f.parent.bootstrapReads != bootstrap || len(f.parent.factoryClients) != 1 {
		t.Fatal("runtime checks repeated bootstrap, constructed another credential, or widened host requests")
	}
}
