package recording

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Pairing(ctx context.Context, user, project, suite, id string) (Pairing, error) {
	token, err := newCompanionToken()
	if err != nil {
		return Pairing{}, errors.New("recording pairing is unavailable")
	}
	var pairing Pairing
	_, _, _, err = s.companionUpdate(ctx, id, func(value Session, _ companionRecord) error {
		if value.Actor != user || value.Project != project || value.Suite != suite {
			return authorization.ErrForbidden
		}
		return nil
	}, func(value *Session, connection *companionRecord) error {
		if !active(value.State) || connection.Instance != "" {
			return storage.ErrConflict
		}
		until := time.Now().UTC().Add(PairingLifetime)
		if value.DeadlineAt.Before(until) {
			until = value.DeadlineAt
		}
		connection.PairHash, connection.PairUntil = companionHash(token), until
		pairing = Pairing{ID: id, Token: token, ExpiresAt: until}
		return nil
	})
	return pairing, err
}

func (s *Service) Pair(ctx context.Context, id string, input PairRequest) (CompanionPacket, error) {
	if !domain.ValidID(input.Instance) || !validCompanionToken(input.DeviceToken) {
		return CompanionPacket{}, authorization.ErrForbidden
	}
	value, connection, _, err := s.companionUpdate(ctx, id, func(value Session, connection companionRecord) error {
		if !matchesCompanionToken(input.PairingToken, connection.PairHash) || !time.Now().Before(connection.PairUntil) {
			return authorization.ErrForbidden
		}
		if connection.Instance != "" && (connection.Instance != input.Instance || !matchesCompanionToken(input.DeviceToken, connection.TokenHash)) {
			return authorization.ErrForbidden
		}
		if !active(value.State) {
			return authorization.ErrForbidden
		}
		return nil
	}, func(value *Session, connection *companionRecord) error {
		if connection.Instance == "" {
			connection.Instance, connection.TokenHash = input.Instance, companionHash(input.DeviceToken)
			value.State = "starting"
		}
		connection.LastSeen = time.Now().UTC()
		return nil
	})
	if err != nil {
		return CompanionPacket{}, err
	}
	return CompanionPacket{Input: value.Input, Project: value.Project, Suite: value.Suite, Control: companionControl(value, connection)}, nil
}

func (s *Service) CompanionControl(ctx context.Context, id, instance, token string) (CompanionControl, error) {
	value, connection, _, err := s.companionUpdate(ctx, id, verifyCompanionDevice(instance, token), func(_ *Session, c *companionRecord) error { c.LastSeen = time.Now().UTC(); return nil })
	if err != nil {
		return CompanionControl{}, err
	}
	return companionControl(value, connection), nil
}

func (s *Service) SubmitCompanion(ctx context.Context, id, instance, token string, input CompanionSnapshot) (CompanionControl, error) {
	if input.Sequence < 1 || input.Sequence > 100000 || len(input.Code) > ContentLimit || !utf8.ValidString(input.Code) || input.State != "recording" && input.State != "ready" && input.State != "failed" || input.State != "failed" && input.Failure != "" || input.Failure != "" && input.Failure != "recorder" && input.Failure != "capture" && input.Failure != "transport" {
		return CompanionControl{}, errors.New("invalid recording companion snapshot")
	}
	hash := companionHash(input)
	value, connection, _, err := s.companionUpdate(ctx, id, verifyCompanionDevice(instance, token), func(value *Session, c *companionRecord) error {
		c.LastSeen = time.Now().UTC()
		if input.Sequence == c.Sequence && hash == c.SnapshotHash {
			return nil
		}
		if !active(value.State) {
			return authorization.ErrForbidden
		}
		if input.Sequence != c.Sequence+1 {
			return storage.ErrConflict
		}
		state, message := input.State, ""
		if state == "failed" {
			message = "The desktop recorder stopped. Review captured steps or reconnect with a new recording session."
		}
		code := input.Code
		if state == "failed" && code == "" {
			code = value.Code
		}
		if state == "ready" && code == "" {
			state, message = "failed", "The recording ended before any Playwright source was captured."
		}
		if c.Action == "cancel" && state != "recording" {
			state = "cancelled"
		}
		if c.Action != "" && state == "recording" {
			state = "stopping"
		}
		value.State, value.Code, value.Error = state, code, message
		c.Sequence, c.SnapshotHash = input.Sequence, hash
		return nil
	})
	if err != nil {
		return CompanionControl{}, err
	}
	return companionControl(value, connection), nil
}

func (s *Service) stopCompanion(ctx context.Context, user, project, suite, id string, cancel bool) (storage.Record, error) {
	_, _, row, err := s.companionUpdate(ctx, id, func(value Session, _ companionRecord) error {
		if value.Actor != user || value.Project != project || value.Suite != suite {
			return authorization.ErrForbidden
		}
		return nil
	}, func(value *Session, c *companionRecord) error {
		if !active(value.State) {
			return nil
		}
		if c.Instance == "" {
			value.State = "cancelled"
			c.Action = "cancel"
			return nil
		}
		if cancel {
			c.Action = "cancel"
		} else if c.Action == "" {
			c.Action = "stop"
		}
		value.State = "stopping"
		return nil
	})
	return row, err
}
