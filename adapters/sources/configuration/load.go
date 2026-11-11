package configuration

import (
	"errors"
	"io"
	"os"

	githubsource "github.com/wtiger001/teststudio/adapters/sources/github"
	"github.com/wtiger001/teststudio/adapters/sources/gitlab"
	"github.com/wtiger001/teststudio/internal/certificates"
	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/networkguard"
	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type Server struct {
	ID      string             `json:"id"`
	URL     string             `json:"url"`
	CAFile  string             `json:"caFile,omitempty"`
	Network networkguard.Scope `json:"network"`
}

func Load(filename string, github bool, credentials secrets.Provider) (map[string]sourceprovider.Provider, error) {
	result := map[string]sourceprovider.Provider{}
	if github {
		result["github"] = &githubsource.Client{Secrets: credentials}
	}
	if filename == "" {
		return result, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.New("Git server configuration is unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil, errors.New("Git server configuration is invalid")
	}
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 65536 {
		return nil, errors.New("Git server configuration cannot be read")
	}
	var servers []Server
	if definitions.DecodeJSON(data, &servers, 65536, 8, 100) != nil || len(servers) < 1 || len(servers) > 20 {
		return nil, errors.New("Git server configuration requires 1 through 20 approved GitLab servers")
	}
	for _, server := range servers {
		if !domain.ValidID(server.ID) || server.ID == "github" || result[server.ID] != nil || server.Network != networkguard.Public && server.Network != networkguard.Private {
			return nil, errors.New("GitLab server requires a distinct identifier and public or private network scope")
		}
		client := &gitlab.Client{Secrets: credentials, BaseURL: server.URL, Network: server.Network}
		if err := client.Validate(sourceprovider.Access{Project: "validation", Repository: "group/repository"}, "heads/main"); err != nil {
			return nil, err
		}
		if server.CAFile != "" {
			roots, _, err := certificates.LoadAuthorities(server.CAFile)
			if err != nil {
				return nil, errors.New("GitLab certificate authority is invalid")
			}
			client.RootCAs = roots
		}
		result[server.ID] = client
	}
	return result, nil
}
