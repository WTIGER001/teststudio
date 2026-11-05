package azuresandbox

import (
	"net/http"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/wtiger001/teststudio/adapters/registry/acr"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type RuntimeOptions struct {
	Settings       Settings
	Exchange       *workerexchange.Service
	Admission      Admission
	ImageAdmission ImageAdmission
	Credentials    func(string) (azcore.TokenCredential, error)
	Transport      http.RoundTripper
	ARMTransport   policy.Transporter
}

type Runtime struct {
	Provider  *Provider
	registry  *acr.Client
	client    *Client
	deadlines *SDKDeadlines
	placement *SDKPlacement
	once      sync.Once
}

func OpenRuntime(o RuntimeOptions) (result *Runtime, resultErr error) {
	s := o.Settings
	if s.Validate() != nil || o.Exchange == nil || o.Exchange.Store == nil || o.Exchange.Authorize == nil || o.Admission == nil || s.Mode == "discovery" && (o.ImageAdmission == nil || o.Exchange.AuthorizeDiscoveryCleanup == nil) {
		return nil, hostedexecution.ErrConfiguration
	}
	credentials := o.Credentials
	if credentials == nil {
		credentials = func(clientID string) (azcore.TokenCredential, error) {
			return azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ID: azidentity.ClientID(clientID)})
		}
	}
	control, err := credentials(s.ControlClientID)
	if err != nil || control == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	reader, err := credentials(s.RegistryClientID)
	if err != nil || reader == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	metadata, err := credentials(s.MetadataClientID)
	if err != nil || metadata == nil {
		return nil, hostedexecution.ErrUnavailable
	}
	r := &Runtime{}
	defer func() {
		if resultErr != nil {
			r.Close()
		}
	}()
	r.client, err = NewClient(s.Worker, control, o.Transport)
	if err != nil {
		return nil, err
	}
	r.registry, err = acr.New(acr.Options{Registry: s.Worker.Registry, Tenant: s.TenantID, RepositoryPrefix: s.Worker.RepositoryPrefix, Credential: reader, BlobRedirectOrigins: s.BlobRedirectOrigins, Transport: o.Transport})
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	r.deadlines, err = NewSDKDeadlines(DeadlineOptions{Config: s.deadlineConfig(), Exchange: o.Exchange, Instances: r.client, Credential: control, Transport: o.ARMTransport})
	if err != nil {
		return nil, err
	}
	r.placement, err = NewSDKPlacement(s, metadata, o.ARMTransport)
	if err != nil {
		return nil, err
	}
	admission := &placementAdmission{placement: r.placement, protection: o.Admission, importProtection: o.ImageAdmission}
	controller, err := NewController(ControllerOptions{Config: s.Worker, Instances: r.client, Deadlines: r.deadlines, Admission: admission, Exchange: o.Exchange, Commands: r.client})
	if err != nil {
		return nil, err
	}
	r.Provider = &Provider{controller: controller}
	images, err := NewImageManager(ImageManagerOptions{Config: s.Worker, PullIdentity: s.PullIdentity, Images: r.client, Exchange: o.Exchange, Admission: admission, AllowImport: s.Mode == "discovery"})
	if err != nil {
		return nil, err
	}
	provider, err := NewProvider(ProviderOptions{Controller: controller, Images: images, Inspector: r.registry, Mode: s.Mode})
	if err != nil {
		return nil, err
	}
	r.Provider = provider
	return r, nil
}

func (r *Runtime) Close() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		if r.Provider != nil {
			r.Provider.Close()
		}
		if r.registry != nil {
			r.registry.Close()
		}
		if r.deadlines != nil {
			r.deadlines.Close()
		}
		if r.placement != nil {
			r.placement.Close()
		}
		if r.client != nil {
			r.client.Close()
		}
	})
}
