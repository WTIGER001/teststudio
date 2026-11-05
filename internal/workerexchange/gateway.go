package workerexchange

import (
	"bytes"
	"context"
	"crypto/subtle"
	"crypto/x509"
	"encoding/json"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/apicredential"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
)

type GatewaySettings struct {
	Origin               string `json:"origin"`
	CertificateAuthority []byte `json:"certificateAuthority,omitempty"`
}

type GatewayRecord struct {
	Settings     GatewaySettings `json:"settings"`
	Instance     string          `json:"instance,omitempty"`
	Bootstrapped bool            `json:"bootstrapped,omitempty"`
}

type GatewayTicket struct {
	Execution string
	Token     string `json:"-"`
	Deadline  time.Time
}

func (t GatewayTicket) String() string   { return "gateway initialization: " + t.Execution }
func (t GatewayTicket) GoString() string { return t.String() }

func gatewayTarget(record Record) (*browserpolicy.Target, error) {
	var input execution.CodedInput
	if record.Operation != "" || !record.DispatchGated || !domain.ValidID(record.Reference.Gateway) || json.Unmarshal(record.Input, &input) != nil || input.Browser == nil || browserpolicy.ValidateTarget(*input.Browser) != nil {
		return nil, ErrInvalid
	}
	return input.Browser, nil
}

func (settings GatewaySettings) valid() bool {
	origin, err := browserpolicy.Origin(settings.Origin)
	return err == nil && origin == settings.Origin && strings.HasPrefix(origin, "https://") && len(settings.CertificateAuthority) <= 64<<10 && (len(settings.CertificateAuthority) == 0 || x509.NewCertPool().AppendCertsFromPEM(settings.CertificateAuthority))
}

func (s *Service) gatewayAuthority(ctx context.Context, record Record) ([]storage.Record, error) {
	if record.Revoked || record.Submission != nil || !time.Now().Before(record.Deadline) || s.Authorize == nil {
		return nil, ErrDenied
	}
	access, err := s.Authorize(ctx, record.Reference)
	if err != nil || access.Deadline.Before(record.Deadline) {
		return nil, ErrDenied
	}
	return access.Records, nil
}

func (s *Service) PrepareGateway(ctx context.Context, ref execution.Reference, settings GatewaySettings) (GatewayTicket, error) {
	for range 8 {
		record, row, err := s.owned(ctx, ref)
		if err != nil {
			return GatewayTicket{}, ErrUnavailable
		}
		guards, err := s.gatewayAuthority(ctx, record)
		if err != nil {
			return GatewayTicket{}, err
		}
		_, err = gatewayTarget(record)
		if err != nil || !settings.valid() {
			return GatewayTicket{}, ErrInvalid
		}
		if record.Gateway != nil {
			if record.Gateway.Settings.Origin != settings.Origin || !bytes.Equal(record.Gateway.Settings.CertificateAuthority, settings.CertificateAuthority) {
				return GatewayTicket{}, ErrConflict
			}
		} else {
			if record.DispatchReady || record.Instance != "" || record.Submission != nil {
				return GatewayTicket{}, ErrConflict
			}
			record.Gateway = &GatewayRecord{Settings: settings}
		}
		token, err := s.scopedToken(ctx, record, "gateway-initialization")
		if err != nil {
			return GatewayTicket{}, err
		}
		if err := s.save(ctx, row, record, guards); err == ErrConflict {
			continue
		} else if err != nil {
			return GatewayTicket{}, err
		}
		return GatewayTicket{Execution: ref.ID, Token: token, Deadline: record.Deadline}, nil
	}
	return GatewayTicket{}, ErrConflict
}

func (s *Service) BindGateway(ctx context.Context, ref execution.Reference, instance string) error {
	if !domain.ValidID(instance) {
		return ErrInvalid
	}
	for range 8 {
		record, row, err := s.owned(ctx, ref)
		if err != nil {
			return ErrUnavailable
		}
		guards, err := s.gatewayAuthority(ctx, record)
		if err != nil {
			return err
		}
		if record.Gateway == nil || record.Gateway.Instance != "" && record.Gateway.Instance != instance {
			return ErrConflict
		}
		record.Gateway.Instance = instance
		if err := s.save(ctx, row, record, guards); err == ErrConflict {
			continue
		} else {
			return err
		}
	}
	return ErrConflict
}

func (s *Service) GatewayInput(ctx context.Context, id, token, instance string) (browsergateway.Configuration, error) {
	if !domain.ValidID(id) || !domain.ValidID(instance) || len(token) != 43 {
		return browsergateway.Configuration{}, ErrDenied
	}
	for range 8 {
		row, err := s.Store.Get(ctx, Bucket, id)
		if err != nil {
			return browsergateway.Configuration{}, ErrDenied
		}
		record, err := storage.Decode[Record](row)
		if err != nil || record.Reference.ID != id || record.Reference.Project != row.Project || record.Gateway == nil {
			return browsergateway.Configuration{}, ErrDenied
		}
		expected, err := s.scopedToken(ctx, record, "gateway-initialization")
		if err != nil {
			return browsergateway.Configuration{}, ErrUnavailable
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			return browsergateway.Configuration{}, ErrDenied
		}
		guards, err := s.gatewayAuthority(ctx, record)
		if err != nil {
			return browsergateway.Configuration{}, err
		}
		if record.Gateway.Instance == "" {
			return browsergateway.Configuration{}, ErrUnavailable
		}
		if record.Gateway.Instance != instance {
			return browsergateway.Configuration{}, ErrDenied
		}
		target, err := gatewayTarget(record)
		if err != nil {
			return browsergateway.Configuration{}, err
		}
		credential, err := apicredential.Resolve(ctx, s.Secrets, record.Reference.Project, record.CredentialReference, *target)
		if err != nil {
			return browsergateway.Configuration{}, ErrUnavailable
		}
		connection, err := s.gatewayConnection(ctx, record)
		if err != nil {
			return browsergateway.Configuration{}, err
		}
		record.Gateway.Bootstrapped = true
		if err := s.save(ctx, row, record, guards); err == ErrConflict {
			continue
		} else if err != nil {
			return browsergateway.Configuration{}, err
		}
		return browsergateway.Configuration{Target: *target, Credential: credential, Remote: &connection.Session}, nil
	}
	return browsergateway.Configuration{}, ErrConflict
}

func (s *Service) GatewayConnection(ctx context.Context, id, token, instance string) (browsergateway.RemoteConnection, error) {
	record, row, guards, err := s.authenticated(ctx, id, token, instance)
	if err != nil {
		return browsergateway.RemoteConnection{}, err
	}
	if record.Instance != instance || record.Gateway == nil || !record.Gateway.Bootstrapped || record.Gateway.Instance == "" || record.Submission != nil {
		return browsergateway.RemoteConnection{}, ErrDenied
	}
	connection, err := s.gatewayConnection(ctx, record)
	if err != nil {
		return browsergateway.RemoteConnection{}, err
	}
	if err := s.save(ctx, row, record, guards); err != nil {
		return browsergateway.RemoteConnection{}, err
	}
	return connection, nil
}

func (s *Service) gatewayConnection(ctx context.Context, record Record) (browsergateway.RemoteConnection, error) {
	token, err := s.scopedToken(ctx, record, "gateway-transport")
	if err != nil {
		return browsergateway.RemoteConnection{}, err
	}
	return browsergateway.RemoteConnection{Origin: record.Gateway.Settings.Origin, CertificateAuthority: record.Gateway.Settings.CertificateAuthority, Session: browsergateway.RemoteSession{Execution: record.Reference.ID, Token: token, Deadline: record.Deadline}}, nil
}
