package ecsworker

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ecrFixture struct {
	calls int
	value *ecr.GetAuthorizationTokenOutput
	err   error
	t     *testing.T
}

func (f *ecrFixture) GetAuthorizationToken(ctx context.Context, input *ecr.GetAuthorizationTokenInput, options ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error) {
	settings := ecr.Options{}
	for _, option := range options {
		option(&settings)
	}
	if len(input.RegistryIds) != 1 || input.RegistryIds[0] != "123456789012" || settings.RetryMaxAttempts != 1 {
		f.t.Fatal("unscoped or retried authorization")
	}
	f.calls++
	return copyValue(f.value), f.err
}

func authFixture(t *testing.T) (*registryAuth, *ecrFixture, *aws.Credentials, *time.Time) {
	t.Helper()
	now := time.Now().UTC()
	credentials := aws.Credentials{AccessKeyID: "fixture-access", SecretAccessKey: "fixture-secret", SessionToken: "fixture-session", CanExpire: true, Expires: now.Add(time.Hour)}
	f := &ecrFixture{t: t, value: &ecr.GetAuthorizationTokenOutput{AuthorizationData: []types.AuthorizationData{{ProxyEndpoint: aws.String("https://123456789012.dkr.ecr.us-east-1.amazonaws.com"), AuthorizationToken: aws.String(base64.StdEncoding.EncodeToString([]byte("AWS:registry-password-canary"))), ExpiresAt: aws.Time(now.Add(12 * time.Hour))}}}}
	auth := &registryAuth{client: f, credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) { return credentials, nil }), account: "123456789012", origin: "https://123456789012.dkr.ecr.us-east-1.amazonaws.com", now: func() time.Time { return now }}
	return auth, f, &credentials, &now
}

func TestECRCredentialsCoalesceBoundCacheAndRotateWithSDKIdentity(t *testing.T) {
	auth, f, current, now := authFixture(t)
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			value, err := auth.Credential(t.Context())
			if err != nil || value.Username != "AWS" || value.Password != "registry-password-canary" {
				t.Error("valid ECR credentials unavailable", err)
			}
		})
	}
	group.Wait()
	if f.calls != 1 || auth.validUntil.Sub(*now) != 5*time.Minute {
		t.Fatal("concurrent authorization did not use its bounded cache")
	}
	current.SessionToken = "rotated-session"
	if _, err := auth.Credential(t.Context()); err != nil || f.calls != 2 {
		t.Fatal("credential rotation reused old registry authority", err)
	}
	*now = now.Add(6 * time.Minute)
	if _, err := auth.Credential(t.Context()); err != nil || f.calls != 3 {
		t.Fatal("expired cache was not refreshed", err)
	}
	current.Expires = now.Add(time.Second)
	auth.validUntil = time.Time{}
	if _, err := auth.Credential(t.Context()); err != nil || !auth.validUntil.Equal(current.Expires) {
		t.Fatal("registry cache outlived workload credentials", err)
	}
	*now = now.Add(2 * time.Second)
	if _, err := auth.Credential(t.Context()); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("expired workload credentials retained cached registry authority", err)
	}
}

func TestECRCredentialsRejectPrivateErrorsAndUnboundAuthorization(t *testing.T) {
	for name, mutate := range map[string]func(*ecrFixture){
		"wrong-registry": func(f *ecrFixture) {
			f.value.AuthorizationData[0].ProxyEndpoint = aws.String("https://999999999999.dkr.ecr.us-east-1.amazonaws.com")
		},
		"ambiguous": func(f *ecrFixture) {
			f.value.AuthorizationData = append(f.value.AuthorizationData, f.value.AuthorizationData[0])
		},
		"expired":       func(f *ecrFixture) { f.value.AuthorizationData[0].ExpiresAt = aws.Time(time.Now().Add(-time.Minute)) },
		"invalid-token": func(f *ecrFixture) { f.value.AuthorizationData[0].AuthorizationToken = aws.String("not-base64") },
		"wrong-user": func(f *ecrFixture) {
			f.value.AuthorizationData[0].AuthorizationToken = aws.String(base64.StdEncoding.EncodeToString([]byte("operator:canary")))
		},
		"header-injection": func(f *ecrFixture) {
			f.value.AuthorizationData[0].AuthorizationToken = aws.String(base64.StdEncoding.EncodeToString([]byte("AWS:canary\r\n")))
		},
		"oversized": func(f *ecrFixture) {
			f.value.AuthorizationData[0].AuthorizationToken = aws.String(strings.Repeat("a", 97<<10))
		},
		"private-error": func(f *ecrFixture) { f.err = errors.New("provider-password-canary") },
	} {
		t.Run(name, func(t *testing.T) {
			auth, f, _, _ := authFixture(t)
			mutate(f)
			value, err := auth.Credential(t.Context())
			if err == nil || strings.Contains(err.Error(), "canary") || value.Password != "" {
				t.Fatal("invalid authorization passed or exposed private data", err)
			}
		})
	}
	auth, f, _, _ := authFixture(t)
	if _, err := auth.Credential(t.Context()); err != nil {
		t.Fatal(err)
	}
	auth.validUntil = time.Time{}
	f.err = errors.New("private-canary")
	if _, err := auth.Credential(t.Context()); err == nil || auth.cached.Password != "" {
		t.Fatal("failed refresh retained stale authority")
	}
}
