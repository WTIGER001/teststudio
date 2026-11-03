package oidc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
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
	GroupMappings        map[string]string `json:"groupMappings,omitempty"`
	MembershipTTLSeconds int               `json:"membershipTtlSeconds,omitempty"`
}

type Service struct {
	Store         *storage.Store
	Directory     *identity.Directory
	Sessions      *sessions.Service
	Origin        string
	Client        *http.Client
	ResolveSecret func(context.Context, string) (string, error)
}

func ValidateProvider(p Provider) error {
	if !domain.ValidID(p.ID) || p.Name == "" || p.ClientID == "" {
		return errors.New("provider identifier, name, and client identifier are required")
	}
	u, err := url.Parse(p.Issuer)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("invalid OIDC issuer")
	}
	ip := net.ParseIP(u.Hostname())
	local := u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return errors.New("OIDC issuer requires HTTPS outside loopback")
	}
	if len(p.GroupMappings) > 40 || (len(p.GroupMappings) > 0 && p.GroupClaim == "") {
		return errors.New("group mappings require a claim name and at most 40 destination groups")
	}
	if p.MembershipTTLSeconds < 0 || p.MembershipTTLSeconds > 3600 {
		return errors.New("membership lifetime must not exceed one hour")
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
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	remote, err := coreoidc.NewProvider(coreoidc.ClientContext(ctx, client), p.Issuer)
	if err != nil {
		return p, nil, oauth2.Config{}, err
	}
	secret := ""
	if p.SecretRef != "" {
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
		if err != nil {
			return p, nil, oauth2.Config{}, err
		}
	}
	scopes := []string{coreoidc.ScopeOpenID, "profile", "email"}
	if len(p.Scopes) > 0 {
		scopes = append([]string{coreoidc.ScopeOpenID}, p.Scopes...)
	}
	cfg := oauth2.Config{ClientID: p.ClientID, ClientSecret: secret, Endpoint: remote.Endpoint(), Scopes: scopes, RedirectURL: s.Origin + "/auth/callback/" + p.ID}
	return p, remote, cfg, nil
}
