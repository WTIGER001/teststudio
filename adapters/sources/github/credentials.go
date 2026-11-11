package github

import (
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"

	"github.com/wtiger001/teststudio/internal/definitions"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

func (c *Client) CredentialInfo(ctx context.Context, a sourceprovider.Access) (sourceprovider.CredentialInfo, error) {
	info := sourceprovider.CredentialInfo{Renewal: "automatic installation token renewal"}
	if a.Credential == "" {
		info.Renewal = "public access"
		return info, nil
	}
	if c.Secrets == nil {
		return info, errors.New("GitHub App credential is unavailable")
	}
	secret, err := c.Secrets.Resolve(ctx, a.Project, a.Credential, sourceprovider.CredentialPurpose)
	if err != nil {
		return info, errors.New("GitHub App credential is unavailable")
	}
	var credential appCredential
	if definitions.DecodeJSON(secret.Bytes, &credential, 65536, 8, 100) != nil {
		return info, errors.New("GitHub App credential is invalid")
	}
	block, _ := pem.Decode([]byte(credential.PrivateKey))
	if block == nil {
		return info, errors.New("GitHub App key is invalid")
	}
	var public any
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		public = &key.PublicKey
	} else if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if key, ok := key.(crypto.Signer); ok {
			public = key.Public()
		}
	}
	if public != nil {
		data, err := x509.MarshalPKIXPublicKey(public)
		if err == nil {
			sum := sha256.Sum256(data)
			info.Fingerprint = hex.EncodeToString(sum[:])
		}
	}
	return info, nil
}
