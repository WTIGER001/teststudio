package oidc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/wtiger001/teststudio/internal/accountlinks"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/identity"
	"github.com/wtiger001/teststudio/internal/sessions"
	"github.com/wtiger001/teststudio/internal/storage"
	"golang.org/x/oauth2"
)

type Provider struct {
	revision             int64
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Issuer               string            `json:"issuer"`
	ClientID             string            `json:"clientId"`
	SecretRef            string            `json:"secretRef,omitempty"`
	Scopes               []string          `json:"scopes"`
	Disabled             bool              `json:"disabled"`
	AutoProvision        bool              `json:"autoProvision"`
	GroupClaim           string            `json:"groupClaim,omitempty"`
	GroupMappings        map[string]string `json:"groupMappings"`
	LogoutEndpoint       string            `json:"logoutEndpoint,omitempty"`
	MembershipTTLSeconds int               `json:"membershipTtlSeconds"`
}

type Service struct {
	ProjectAccess ProjectAccess
	Links         *accountlinks.Service
	Store         *storage.Store
	Directory     *identity.Directory
	Sessions      *sessions.Service
	Origin        string
	Client        *http.Client
	ResolveSecret func(context.Context, string) (string, error)
	SecretRefs    []string
}

var environmentReference = regexp.MustCompile(`^env:[A-Za-z_][A-Za-z0-9_]{0,127}$`)

func ValidSecretReference(value string) bool { return environmentReference.MatchString(value) }

func ValidateProvider(p Provider) error {
	if !domain.ValidID(p.ID) || strings.TrimSpace(p.Name) == "" || len(p.Name) > 200 || p.ClientID == "" || len(p.ClientID) > 512 || len(p.Issuer) > 2048 {
		return errors.New("provider identifier, name, and client identifier are required")
	}
	u, err := url.Parse(p.Issuer)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return errors.New("invalid OIDC issuer")
	}
	ip := net.ParseIP(u.Hostname())
	local := u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return errors.New("OIDC issuer requires HTTPS outside loopback")
	}
	if p.LogoutEndpoint != "" {
		if err := validateLogoutEndpoint(p.LogoutEndpoint); err != nil {
			return err
		}
	}
	if len(p.GroupMappings) > 40 || (len(p.GroupMappings) > 0 && p.GroupClaim == "") {
		return errors.New("group mappings require a claim name and at most 40 destination groups")
	}
	if p.MembershipTTLSeconds < 0 || p.MembershipTTLSeconds > 3600 {
		return errors.New("membership lifetime must not exceed one hour")
	}
	if len(p.GroupClaim) > 256 || len(p.Scopes) > 32 || (p.SecretRef != "" && !ValidSecretReference(p.SecretRef)) {
		return errors.New("provider claim, scopes, or credential reference is invalid")
	}
	seen := map[string]bool{}
	for _, scope := range p.Scopes {
		if scope == "" || len(scope) > 256 || seen[scope] || strings.ContainsAny(scope, "\"\\") || strings.IndexFunc(scope, func(r rune) bool { return r < 0x21 || r > 0x7e }) >= 0 {
			return errors.New("provider scopes require unique bounded printable tokens")
		}
		seen[scope] = true
	}
	for external, group := range p.GroupMappings {
		if strings.TrimSpace(external) == "" || len(external) > 512 || !domain.ValidID(group) {
			return errors.New("each external group mapping requires a bounded claim value and destination group identifier")
		}
	}
	return nil
}

func (s *Service) config(ctx context.Context, id string) (Provider, *coreoidc.Provider, oauth2.Config, error) {
	r, err := s.Store.Get(ctx, "providers", id)
	if err != nil {
		return Provider{}, nil, oauth2.Config{}, err
	}
	p, err := storage.Decode[Provider](r)
	if err != nil {
		return p, nil, oauth2.Config{}, err
	}
	if p.ID != id || r.Project != "" {
		return p, nil, oauth2.Config{}, errors.New("provider record has invalid ownership")
	}
	if err := ValidateProvider(p); err != nil {
		return p, nil, oauth2.Config{}, err
	}
	if p.Disabled {
		return p, nil, oauth2.Config{}, errors.New("provider is disabled")
	}
	p.revision = r.Revision
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	client := s.httpClient()
	remote, err := coreoidc.NewProvider(coreoidc.ClientContext(ctx, client), p.Issuer)
	if err != nil {
		return p, nil, oauth2.Config{}, errors.New("OIDC provider discovery failed")
	}
	var metadata coreoidc.ProviderConfig
	if err := remote.Claims(&metadata); err != nil {
		return p, nil, oauth2.Config{}, errors.New("OIDC provider metadata is invalid")
	}
	for _, endpoint := range []string{metadata.AuthURL, metadata.TokenURL, metadata.JWKSURL} {
		if err := protocolEndpoint(endpoint); err != nil {
			return p, nil, oauth2.Config{}, err
		}
	}
	secret := ""
	if p.SecretRef != "" {
		if !slices.Contains(s.SecretRefs, p.SecretRef) {
			return p, nil, oauth2.Config{}, errors.New("OIDC credential reference is not approved for this installation")
		}
		if s.ResolveSecret != nil {
			secret, err = s.ResolveSecret(ctx, p.SecretRef)
		} else if strings.HasPrefix(p.SecretRef, "env:") {
			secret = os.Getenv(strings.TrimPrefix(p.SecretRef, "env:"))
			if secret == "" {
				err = errors.New("OIDC credential reference is unavailable")
			}
		} else {
			err = errors.New("unsupported OIDC credential reference")
		}
		if err != nil || secret == "" {
			return p, nil, oauth2.Config{}, errors.New("OIDC credential reference is unavailable")
		}
	}
	scopes := []string{coreoidc.ScopeOpenID, "profile", "email"}
	if len(p.Scopes) > 0 {
		scopes = []string{coreoidc.ScopeOpenID}
		for _, scope := range p.Scopes {
			if scope != coreoidc.ScopeOpenID {
				scopes = append(scopes, scope)
			}
		}
	}
	cfg := oauth2.Config{ClientID: p.ClientID, ClientSecret: secret, Endpoint: remote.Endpoint(), Scopes: scopes, RedirectURL: s.Origin + "/auth/callback/" + p.ID}
	return p, remote, cfg, nil
}
