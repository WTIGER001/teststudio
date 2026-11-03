package sourceprovider

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"
)

const CredentialPurpose = "source-repository"
const WebhookPurpose = "source-webhook"

type Access struct {
	Project      string
	Repository   string
	RepositoryID string
	Credential   string
}

type Revision struct {
	RepositoryID string `json:"repositoryId"`
	Repository   string `json:"repository"`
	URL          string `json:"url"`
	Reference    string `json:"reference"`
	Commit       string `json:"commit"`
	Private      bool   `json:"private"`
}

type Archive struct {
	Content []byte
	Root    string
}

type Provider interface {
	Validate(Access, string) error
	Resolve(context.Context, Access, string) (Revision, error)
	Archive(context.Context, Access, Revision) (Archive, error)
}

var commit = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)

func ValidCommit(value string) bool { return commit.MatchString(value) }
func ValidReference(value string) bool {
	if ValidCommit(value) {
		return true
	}
	if !(strings.HasPrefix(value, "heads/") || strings.HasPrefix(value, "tags/")) {
		return false
	}
	_, name, _ := strings.Cut(value, "/")
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return len(value) <= 256 && name != "" && !strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "/") && !strings.HasSuffix(name, "/") && !strings.HasSuffix(name, ".") && !strings.Contains(name, "//") && !strings.Contains(name, "..") && !strings.Contains(name, "@{") && !strings.ContainsAny(name, "\\ ~^:?*[\x00") && !strings.ContainsFunc(name, unicode.IsControl)
}

var ErrUnavailable = errors.New("source provider is unavailable")

type WebhookRequest struct {
	Delivery  string
	Event     string
	Signature string
	Body      []byte
}

type WebhookEvent struct {
	Delivery     string
	Kind         string
	RepositoryID string
	Repository   string
	Reference    string
	Commit       string
	Deleted      bool
}

type WebhookVerifier interface {
	VerifyWebhook(context.Context, Access, string, WebhookRequest) (WebhookEvent, error)
}

var ErrWebhookRejected = errors.New("source webhook authentication or configuration is invalid")
