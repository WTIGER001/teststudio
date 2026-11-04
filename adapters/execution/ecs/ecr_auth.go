package ecsworker

import (
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ECR interface {
	GetAuthorizationToken(context.Context, *ecr.GetAuthorizationTokenInput, ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error)
}

type registryAuth struct {
	client          ECR
	credentials     aws.CredentialsProvider
	account, origin string
	now             func() time.Time
	mu              sync.Mutex
	cached          oci.Credential
	identity        string
	validUntil      time.Time
}

func registryOrigin(c Config) string {
	return "https://" + c.Account + ".dkr.ecr." + c.Region + ".amazonaws.com"
}

func oneECR(options *ecr.Options) { options.RetryMaxAttempts = 1 }

func (a *registryAuth) Credential(ctx context.Context) (oci.Credential, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return oci.Credential{}, hostedexecution.ErrUnavailable
	}
	current, err := a.credentials.Retrieve(ctx)
	if err != nil || !current.HasKeys() || current.CanExpire && !current.Expires.After(a.now()) {
		return oci.Credential{}, hostedexecution.ErrUnavailable
	}
	identity := fingerprint([]string{current.AccessKeyID, current.SecretAccessKey, current.SessionToken})
	if a.identity == identity && a.now().Before(a.validUntil) {
		return a.cached, nil
	}
	a.cached = oci.Credential{}
	a.identity = ""
	a.validUntil = time.Time{}
	result, err := a.client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{RegistryIds: []string{a.account}}, oneECR)
	if err != nil || result == nil || len(result.AuthorizationData) != 1 {
		return oci.Credential{}, hostedexecution.ErrUnavailable
	}
	value := result.AuthorizationData[0]
	now := a.now()
	if ctx.Err() != nil || current.CanExpire && !current.Expires.After(now) || aws.ToString(value.ProxyEndpoint) != a.origin || value.ExpiresAt == nil || !value.ExpiresAt.After(now.Add(2*time.Minute)) || len(aws.ToString(value.AuthorizationToken)) > 96<<10 {
		return oci.Credential{}, hostedexecution.ErrUnavailable
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(aws.ToString(value.AuthorizationToken))
	username, password, ok := strings.Cut(string(raw), ":")
	if err != nil || !ok || username != "AWS" || password == "" || len(password) > 65536 || strings.ContainsAny(password, "\x00\r\n") {
		return oci.Credential{}, hostedexecution.ErrUnavailable
	}
	a.cached = oci.Credential{Username: username, Password: password}
	a.identity = identity
	a.validUntil = minTime(value.ExpiresAt.Add(-2*time.Minute), now.Add(5*time.Minute))
	if current.CanExpire {
		a.validUntil = minTime(a.validUntil, current.Expires)
	}
	return a.cached, nil
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}
