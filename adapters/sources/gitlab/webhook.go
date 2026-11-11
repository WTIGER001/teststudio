package gitlab

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func (c *Client) VerifyWebhook(ctx context.Context, a sourceprovider.Access, credential string, r sourceprovider.WebhookRequest) (sourceprovider.WebhookEvent, error) {
	reject := sourceprovider.ErrWebhookRejected
	if c.Secrets == nil || !domain.ValidID(a.Project) || !domain.ValidID(credential) || !domain.ValidID(r.Delivery) || len(r.Body) == 0 || len(r.Body) > 2<<20 {
		return sourceprovider.WebhookEvent{}, reject
	}
	secret, err := c.Secrets.Resolve(ctx, a.Project, credential, sourceprovider.WebhookPurpose)
	if err != nil || len(secret.Bytes) < 32 || len(secret.Bytes) > 4096 || subtle.ConstantTimeCompare(secret.Bytes, []byte(r.Signature)) != 1 {
		return sourceprovider.WebhookEvent{}, reject
	}
	var p struct {
		Kind    string `json:"object_kind"`
		Ref     string `json:"ref"`
		After   string `json:"after"`
		Project struct {
			ID   int64  `json:"id"`
			Name string `json:"path_with_namespace"`
		} `json:"project"`
	}
	if json.Unmarshal(r.Body, &p) != nil || p.Project.ID < 1 || !validRepository(p.Project.Name) {
		return sourceprovider.WebhookEvent{}, reject
	}
	e := sourceprovider.WebhookEvent{Delivery: r.Delivery, RepositoryID: strconv.FormatInt(p.Project.ID, 10), Repository: p.Project.Name, Kind: p.Kind}
	if r.Event == "Push Hook" || r.Event == "Tag Push Hook" {
		e.Kind = "push"
		e.Reference = strings.TrimPrefix(p.Ref, "refs/")
		e.Commit = p.After
		e.Deleted = p.After == strings.Repeat("0", 40)
		if !strings.HasPrefix(p.Ref, "refs/") || !sourceprovider.ValidReference(e.Reference) || !sourceprovider.ValidCommit(p.After) {
			return sourceprovider.WebhookEvent{}, reject
		}
	}
	return e, nil
}
