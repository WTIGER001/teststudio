package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/secrets"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func (c *Client) CredentialInfo(ctx context.Context, a sourceprovider.Access) (sourceprovider.CredentialInfo, error) {
	v, err := c.access(ctx, a)
	if err != nil {
		return sourceprovider.CredentialInfo{}, err
	}
	info := sourceprovider.CredentialInfo{ExpiresAt: v.ExpiresAt, Renewal: "operator replacement"}
	if p, ok := c.Secrets.(secrets.Replacer); ok && p.Replaceable(a.Project, a.Credential, sourceprovider.CredentialPurpose) && v.TokenType == "project" {
		info.CanRotate = true
		info.Renewal = "project token self-rotation"
	}
	return info, nil
}
func (c *Client) RotateCredential(ctx context.Context, a sourceprovider.Access) error {
	p, ok := c.Secrets.(secrets.Replacer)
	if !ok || !p.Replaceable(a.Project, a.Credential, sourceprovider.CredentialPurpose) {
		return errors.New("GitLab token rotation requires a writable mounted-file secret binding")
	}
	old, err := c.Secrets.Resolve(ctx, a.Project, a.Credential, sourceprovider.CredentialPurpose)
	if err != nil {
		return errors.New("GitLab credential is unavailable")
	}
	v, err := c.access(ctx, a)
	if err != nil {
		return err
	}
	if v.TokenType != "project" {
		return errors.New("self-rotation requires an approved project access token binding")
	}
	repo, err := c.repository(ctx, a)
	if err != nil {
		return err
	}
	data, err := c.request(ctx, a, "POST", "/projects/"+strconv.FormatInt(repo.ID, 10)+"/access_tokens/self/rotate", strings.NewReader("{}"), 64<<10)
	if err != nil {
		return errors.New("GitLab rotation was not confirmed; replace the credential before another rotation")
	}
	var rotated struct {
		Token   string `json:"token"`
		Expires string `json:"expires_at"`
	}
	if json.Unmarshal(data, &rotated) != nil || len(rotated.Token) < 1 || len(rotated.Token) > 4096 {
		return errors.New("GitLab rotation returned an invalid credential; operator replacement is required")
	}
	for _, b := range rotated.Token {
		if b <= 32 || b >= 127 {
			return errors.New("GitLab rotation returned an invalid credential; operator replacement is required")
		}
	}
	v.Token = rotated.Token
	if rotated.Expires != "" {
		expiry, err := time.Parse("2006-01-02", rotated.Expires)
		if err != nil {
			return errors.New("GitLab rotation expiry is invalid; operator replacement is required")
		}
		v.ExpiresAt = &expiry
	}
	replacement, err := json.Marshal(v)
	if err != nil {
		return errors.New("GitLab replacement credential could not be encoded")
	}
	checkpoint, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err = p.Replace(checkpoint, a.Project, a.Credential, sourceprovider.CredentialPurpose, old.Bytes, replacement); err != nil {
		return errors.New("GitLab rotated its token but replacement storage was not confirmed; operator recovery is required")
	}
	return nil
}
