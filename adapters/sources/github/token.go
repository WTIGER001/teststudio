package github

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func (c *Client) repositoryCredential(ctx context.Context, access sourceprovider.Access) (repositoryCredential, error) {
	var credential repositoryCredential
	if c.Secrets == nil {
		return credential, errors.New("GitHub credential reference is unavailable")
	}
	secret, err := c.Secrets.Resolve(ctx, access.Project, access.Credential, sourceprovider.CredentialPurpose)
	if err != nil {
		return credential, errors.New("GitHub credential reference is unavailable")
	}
	if definitions.DecodeJSON(secret.Bytes, &credential, 65536, 8, 100) != nil {
		return repositoryCredential{}, errors.New("GitHub credential configuration is invalid")
	}
	allowed := false
	if len(credential.Repositories) < 1 || len(credential.Repositories) > 500 {
		return repositoryCredential{}, errors.New("GitHub credential requires an approved repository list")
	}
	for _, repository := range credential.Repositories {
		if !repositoryName.MatchString(repository) {
			return repositoryCredential{}, errors.New("GitHub credential repository allowlist is invalid")
		}
		allowed = allowed || strings.EqualFold(repository, access.Repository)
	}
	if !allowed {
		return repositoryCredential{}, errors.New("repository is not approved for this GitHub credential")
	}
	if access.Write && !credential.Write {
		return repositoryCredential{}, errors.New("GitHub credential is not approved for metadata writes")
	}
	if credential.Token != "" {
		if credential.ClientID != "" || credential.Installation != 0 || credential.PrivateKey != "" || len(credential.Token) > 4096 {
			return repositoryCredential{}, errors.New("GitHub token credential configuration is invalid")
		}
		for _, character := range credential.Token {
			if character <= 32 || character >= 127 {
				return repositoryCredential{}, errors.New("GitHub token contains invalid header characters")
			}
		}
		if credential.ExpiresAt != nil && !time.Now().Before(*credential.ExpiresAt) {
			return repositoryCredential{}, errors.New("GitHub token has expired; replace its approved credential reference")
		}
		return credential, nil
	}
	if credential.ExpiresAt != nil || len(credential.ClientID) < 1 || len(credential.ClientID) > 128 || credential.Installation <= 0 || len(credential.PrivateKey) > 16384 {
		return repositoryCredential{}, errors.New("GitHub credential requires a nonempty API token or valid App configuration")
	}
	for _, character := range credential.ClientID {
		if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '.' || character == '_' || character == '-') {
			return repositoryCredential{}, errors.New("GitHub App client identifier is invalid")
		}
	}
	return credential, nil
}
