package sourceprovider

import (
	"context"
	"time"
)

type CredentialInfo struct {
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	Renewal     string     `json:"renewal"`
	Fingerprint string     `json:"fingerprint,omitempty"`
	CanRotate   bool       `json:"canRotate"`
}
type CredentialInspector interface {
	CredentialInfo(context.Context, Access) (CredentialInfo, error)
}
type CredentialRotator interface {
	RotateCredential(context.Context, Access) error
}
