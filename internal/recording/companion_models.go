package recording

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"
)

const companionBucket = "recording-companions"
const PairingLifetime = 5 * time.Minute
const CompanionLease = 20 * time.Second

type Pairing struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (p Pairing) String() string   { return "recording pairing: " + p.ID }
func (p Pairing) GoString() string { return p.String() }

type PairRequest struct {
	PairingToken string `json:"pairingToken"`
	Instance     string `json:"instance"`
	DeviceToken  string `json:"deviceToken"`
}

func (p PairRequest) String() string   { return "recording companion pairing" }
func (p PairRequest) GoString() string { return p.String() }

type CompanionPacket struct {
	Input   Input            `json:"input"`
	Project string           `json:"projectId"`
	Suite   string           `json:"suiteId"`
	Control CompanionControl `json:"control"`
}

type CompanionControl struct {
	State      string    `json:"state"`
	Action     string    `json:"action,omitempty"`
	DeadlineAt time.Time `json:"deadlineAt"`
	Sequence   uint64    `json:"sequence"`
}

type CompanionSnapshot struct {
	Sequence uint64 `json:"sequence"`
	State    string `json:"state"`
	Code     string `json:"code,omitempty"`
	Failure  string `json:"failure,omitempty"`
}

type companionRecord struct {
	Binding      string    `json:"binding"`
	PairHash     string    `json:"pairHash,omitempty"`
	PairUntil    time.Time `json:"pairUntil,omitempty"`
	Instance     string    `json:"instance,omitempty"`
	TokenHash    string    `json:"tokenHash,omitempty"`
	Sequence     uint64    `json:"sequence"`
	SnapshotHash string    `json:"snapshotHash,omitempty"`
	LastSeen     time.Time `json:"lastSeen,omitempty"`
	Action       string    `json:"action,omitempty"`
}

func companionHash(value any) string {
	data, _ := json.Marshal(value)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
func companionBinding(s Session) string {
	return companionHash([]any{s.Input, s.Mode, s.Project, s.Suite, s.Actor, s.CreatedAt, s.DeadlineAt})
}
func newCompanionToken() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}
func validCompanionToken(value string) bool {
	data, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && len(data) == 32 && len(value) == 43
}
func matchesCompanionToken(token, hash string) bool {
	return validCompanionToken(token) && len(hash) == 64 && subtle.ConstantTimeCompare([]byte(companionHash(token)), []byte(hash)) == 1
}
func companionControl(s Session, c companionRecord) CompanionControl {
	return CompanionControl{State: s.State, Action: c.Action, DeadlineAt: s.DeadlineAt, Sequence: c.Sequence}
}
