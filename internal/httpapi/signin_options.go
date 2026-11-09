package httpapi

import (
	"net/http"

	authoidc "github.com/wtiger001/teststudio/adapters/auth/oidc"
	"github.com/wtiger001/teststudio/internal/storage"
)

type signInProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type signInOptions struct {
	Providers            []signInProvider `json:"providers"`
	LocalEnabled         bool             `json:"localEnabled"`
	LocalAccountsEnabled bool             `json:"localAccountsEnabled"`
	AutoRedirectProvider string           `json:"autoRedirectProvider"`
}

func (s *Server) signInOptions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.All(r.Context(), "providers", "")
	if err != nil {
		problem(w, err)
		return
	}
	options := signInOptions{Providers: []signInProvider{}, LocalEnabled: s.LocalUser != ""}
	redirects := []string{}
	for _, row := range rows {
		provider, err := storage.Decode[authoidc.Provider](row)
		if err != nil {
			problem(w, err)
			return
		}
		if row.Project != "" || row.ID != provider.ID || provider.Disabled || provider.HiddenFromSignIn {
			continue
		}
		options.Providers = append(options.Providers, signInProvider{ID: provider.ID, Name: provider.Name})
		if provider.LoginBypassEnabled {
			redirects = append(redirects, provider.ID)
		}
	}
	if s.LocalAccounts != nil {
		options.LocalAccountsEnabled, err = s.Auth.Directory.LocalAccountsEnabled(r.Context())
		if err != nil {
			problem(w, err)
			return
		}
	}
	if !options.LocalEnabled && !options.LocalAccountsEnabled && len(redirects) == 1 {
		options.AutoRedirectProvider = redirects[0]
	}
	JSON(w, http.StatusOK, options)
}
