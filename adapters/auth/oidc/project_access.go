package oidc

import (
	"context"
	"time"
)

type SignInIdentity struct {
	UserID           string
	ProviderID       string
	ProviderRevision int64
	Issuer           string
	Subject          string
	Email            string
	EmailVerified    bool
	ExpiresAt        time.Time
}

type ProjectAccess interface {
	ApplyPendingAccess(context.Context, SignInIdentity) error
}
