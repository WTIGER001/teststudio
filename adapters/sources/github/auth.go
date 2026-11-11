package github

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type repositoryCredential struct {
	Token        string     `json:"token,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	Write        bool       `json:"write,omitempty"`
	Repositories []string   `json:"repositories"`
	ClientID     string     `json:"clientId"`
	Installation int64      `json:"installationId"`
	PrivateKey   string     `json:"privateKeyPem"`
}

func (c *Client) sourceToken(ctx context.Context, session *session, access sourceprovider.Access) (string, error) {
	credential, err := c.repositoryCredential(ctx, access)
	if err != nil {
		return "", err
	}
	if credential.Token != "" {
		return credential.Token, nil
	}
	level := "read"
	permissions := map[string]string{"contents": level}
	if access.Write {
		level = "write"
		permissions["contents"] = level
		permissions["pull_requests"] = "write"
	}
	token, err := appJWT(credential, time.Now())
	if err != nil {
		return "", err
	}
	_, name, _ := strings.Cut(access.Repository, "/")
	response, err := session.request(ctx, "POST", session.api+"/app/installations/"+strconv.FormatInt(credential.Installation, 10)+"/access_tokens", map[string]any{"repositories": []string{name}, "permissions": permissions}, token, "application/vnd.github+json")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != 201 {
		return "", statusError("installation authentication", response.StatusCode)
	}
	data, err := readBounded(response.Body, 64<<10)
	if err != nil {
		return "", errors.New("GitHub installation authentication response is invalid")
	}
	var issued struct {
		Token       string            `json:"token"`
		ExpiresAt   time.Time         `json:"expires_at"`
		Permissions map[string]string `json:"permissions"`
	}
	if json.Unmarshal(data, &issued) != nil || len(issued.Token) < 1 || len(issued.Token) > 4096 || !time.Now().Before(issued.ExpiresAt) || issued.ExpiresAt.After(time.Now().Add(65*time.Minute)) || issued.Permissions["contents"] != level {
		return "", errors.New("GitHub installation authentication response is invalid")
	}
	for _, character := range issued.Token {
		if character <= 32 || character >= 127 {
			return "", errors.New("GitHub installation token contains invalid header characters")
		}
	}
	for permission, level := range issued.Permissions {
		if !(permission == "metadata" && level == "read" || permission == "contents" && (level == "read" || access.Write && level == "write") || access.Write && permission == "pull_requests" && level == "write") {
			return "", errors.New("GitHub installation token exceeds its read-only source permissions")
		}
	}
	return issued.Token, nil
}
func appJWT(credential repositoryCredential, now time.Time) (string, error) {
	block, remaining := pem.Decode([]byte(credential.PrivateKey))
	if block == nil || len(strings.TrimSpace(string(remaining))) != 0 {
		return "", errors.New("GitHub App requires an RSA private key")
	}
	var key *rsa.PrivateKey
	if parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		key = parsed
	} else if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		key, _ = parsed.(*rsa.PrivateKey)
	}
	if key == nil || key.N.BitLen() < 2048 || key.Validate() != nil {
		return "", errors.New("GitHub App requires a valid RSA private key of at least 2048 bits")
	}
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{"iat": now.Add(-time.Minute).Unix(), "exp": now.Add(5 * time.Minute).Unix(), "iss": credential.ClientID})
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	hash := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		return "", errors.New("GitHub App assertion could not be signed")
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}
