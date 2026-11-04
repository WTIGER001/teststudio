package acr

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/wtiger001/teststudio/adapters/registry/oci"
)

type Options struct {
	Registry            string
	Tenant              string
	RepositoryPrefix    string
	Publish             bool
	Credential          azcore.TokenCredential
	BlobRedirectOrigins []string
	Transport           http.RoundTripper
	Now                 func() time.Time
}

type Client struct {
	*oci.Client
	auth *authentication
	once sync.Once
}

var registryName = regexp.MustCompile(`^[a-z0-9]{5,50}\.azurecr\.io$`)
var tenantID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var blobName = regexp.MustCompile(`^[a-z0-9]{3,24}\.blob\.core\.windows\.net$`)
var dataRegion = regexp.MustCompile(`^[a-z0-9]+$`)

func New(options Options) (*Client, error) {
	if !registryName.MatchString(options.Registry) || !tenantID.MatchString(options.Tenant) || !oci.RepositoryWithinPrefix(options.RepositoryPrefix, options.RepositoryPrefix) || options.Credential == nil || len(options.BlobRedirectOrigins) > 16 {
		return nil, oci.ErrConfiguration
	}
	origins := map[string]bool{}
	for _, origin := range options.BlobRedirectOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Host != u.Hostname() || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != origin || origins[origin] {
			return nil, oci.ErrConfiguration
		}
		short := strings.TrimSuffix(options.Registry, ".azurecr.io")
		region := strings.TrimSuffix(strings.TrimPrefix(u.Host, short+"."), ".data.azurecr.io")
		if !blobName.MatchString(u.Host) && !(strings.HasPrefix(u.Host, short+".") && strings.HasSuffix(u.Host, ".data.azurecr.io") && dataRegion.MatchString(region)) {
			return nil, oci.ErrConfiguration
		}
		origins[origin] = true
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	transport := newTransport(options, origins)
	auth := &authentication{registry: options.Registry, tenant: options.Tenant, prefix: options.RepositoryPrefix, publish: options.Publish, credential: options.Credential, now: now, slots: make(chan struct{}, 1), entries: map[string]cachedToken{}, http: &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	transport.closed = &auth.closed
	client, err := oci.New(oci.Options{Origin: "https://" + options.Registry, RepositoryPrefix: options.RepositoryPrefix, Architecture: "amd64", TokenURL: "https://" + options.Registry + "/oauth2/token", TokenService: options.Registry, Tokens: auth.Token, Transport: transport, BlobRedirectOrigins: options.BlobRedirectOrigins})
	if err != nil {
		return nil, err
	}
	return &Client{Client: client, auth: auth}, nil
}

func (c *Client) Close() {
	c.once.Do(func() {
		c.auth.closed.Store(true)
		c.auth.slots <- struct{}{}
		c.auth.clear()
		<-c.auth.slots
		c.Client.Close()
	})
}
