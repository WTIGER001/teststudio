package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func (c *Client) VerifyWebhook(ctx context.Context, access sourceprovider.Access, credential string, request sourceprovider.WebhookRequest) (sourceprovider.WebhookEvent, error) {
	var event sourceprovider.WebhookEvent
	reject := sourceprovider.ErrWebhookRejected
	if c.Secrets == nil || !domain.ValidID(access.Project) || !domain.ValidID(credential) || !domain.ValidID(request.Delivery) || len(request.Delivery) > 128 || len(request.Body) == 0 || len(request.Body) > 2<<20 || !strings.HasPrefix(request.Signature, "sha256=") {
		return event, reject
	}
	signature, err := hex.DecodeString(strings.TrimPrefix(request.Signature, "sha256="))
	if err != nil || len(signature) != sha256.Size {
		return event, reject
	}
	secret, err := c.Secrets.Resolve(ctx, access.Project, credential, sourceprovider.WebhookPurpose)
	if err != nil || len(secret.Bytes) < 32 || len(secret.Bytes) > 4096 {
		return event, reject
	}
	mac := hmac.New(sha256.New, secret.Bytes)
	mac.Write(request.Body)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return event, reject
	}
	var payload struct {
		Reference  string `json:"ref"`
		Commit     string `json:"after"`
		Deleted    bool   `json:"deleted"`
		Repository struct {
			ID   int64  `json:"id"`
			Name string `json:"full_name"`
		} `json:"repository"`
	}
	if json.Unmarshal(request.Body, &payload) != nil || payload.Repository.ID < 1 || !repositoryName.MatchString(payload.Repository.Name) {
		return event, reject
	}
	event = sourceprovider.WebhookEvent{Delivery: request.Delivery, Kind: request.Event, RepositoryID: strconv.FormatInt(payload.Repository.ID, 10), Repository: payload.Repository.Name, Deleted: payload.Deleted}
	if request.Event == "push" {
		if !strings.HasPrefix(payload.Reference, "refs/") || !sourceprovider.ValidReference(strings.TrimPrefix(payload.Reference, "refs/")) || !sourceprovider.ValidCommit(payload.Commit) {
			return sourceprovider.WebhookEvent{}, reject
		}
		event.Reference = strings.TrimPrefix(payload.Reference, "refs/")
		event.Commit = payload.Commit
	}
	return event, nil
}
