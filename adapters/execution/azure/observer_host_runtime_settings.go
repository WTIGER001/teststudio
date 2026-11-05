package azureworker

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ObserverHostRuntime struct {
	RuntimeClientID            string `json:"runtimeClientId"`
	DatabaseSecretURL          string `json:"databaseSecretUrl"`
	OIDCConfigurationSecretURL string `json:"oidcConfigurationSecretUrl"`
	OIDCClientOneSecretURL     string `json:"oidcClientOneSecretUrl,omitempty"`
	OIDCClientTwoSecretURL     string `json:"oidcClientTwoSecretUrl,omitempty"`
	BlobServiceURL             string `json:"blobServiceUrl"`
	EvidenceContainer          string `json:"evidenceContainer"`
	UIOrigin                   string `json:"uiOrigin"`
	ArtifactOrigin             string `json:"artifactOrigin"`
	BootstrapIssuer            string `json:"bootstrapIssuer"`
	BootstrapSubject           string `json:"bootstrapSubject"`
	LocalAccounts              bool   `json:"localAccounts"`
	Replicas                   int32  `json:"replicas"`
}

func (r ObserverHostRuntime) canonical(c Config, s ObserverHostSettings) (ObserverHostRuntime, error) {
	if r.Replicas == 0 {
		r.Replicas = 1
	}
	backend := s.Identity.Protection.PrivateDNS.Endpoint.DNS.Firewall.Perimeter.Topology.BackendEndpoint.Backend
	if !observerGUID(r.RuntimeClientID) || r.RuntimeClientID == s.Identity.ClientID || r.RuntimeClientID == backend.RuntimeClientID || !backendBlob.MatchString(r.BlobServiceURL) || !backendContainer.MatchString(r.EvidenceContainer) || strings.Contains(r.EvidenceContainer, "--") || r.Replicas < 1 || r.Replicas > 10 || !observerHostIssuer(r.BootstrapIssuer) || r.BootstrapSubject == "" || len(r.BootstrapSubject) > 512 || strings.IndexFunc(r.BootstrapSubject, unicode.IsControl) >= 0 {
		return r, hostedexecution.ErrConfiguration
	}
	seen := map[string]bool{}
	vault := ""
	for i, secret := range []string{r.DatabaseSecretURL, r.OIDCConfigurationSecretURL, r.OIDCClientOneSecretURL, r.OIDCClientTwoSecretURL} {
		if secret == "" && i > 1 {
			continue
		}
		if !backendSecret.MatchString(secret) || seen[strings.ToLower(secret)] {
			return r, hostedexecution.ErrConfiguration
		}
		u, _ := url.Parse(secret)
		if vault != "" && vault != u.Host {
			return r, hostedexecution.ErrConfiguration
		}
		vault, seen[strings.ToLower(secret)] = u.Host, true
	}
	for _, origin := range []string{r.UIOrigin, r.ArtifactOrigin} {
		config := c
		config.ExchangeOrigin = origin
		if Validate(config) != nil || origin == c.ExchangeOrigin || origin == backend.Exchange.BackendOrigin {
			return r, hostedexecution.ErrConfiguration
		}
	}
	_, name := resourceParts(s.Application)
	ui, _ := url.Parse(r.UIOrigin)
	artifact, _ := url.Parse(r.ArtifactOrigin)
	domain := strings.TrimPrefix(ui.Hostname(), name+".")
	if domain == ui.Hostname() || !strings.Contains(domain, ".") || artifact.Hostname() == ui.Hostname() || !strings.HasSuffix(artifact.Hostname(), "."+domain) || strings.Count(strings.TrimSuffix(artifact.Hostname(), "."+domain), ".") != 0 {
		return r, hostedexecution.ErrConfiguration
	}
	return r, nil
}

func observerHostIssuer(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && len(raw) <= 2048 && u.Scheme == "https" && u.Host == u.Hostname() && hostname.MatchString(u.Hostname()) && strings.Contains(u.Hostname(), ".") && !strings.Contains(u.Hostname(), "..") && u.User == nil && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.Opaque == "" && u.String() == raw && strings.IndexFunc(raw, unicode.IsControl) < 0
}
