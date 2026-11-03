package cliauth

import (
	"crypto/subtle"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/sessions"
)

var ErrInvalid = errors.New("CLI sign-in is invalid or expired; start it again in the terminal")
var challenge = regexp.MustCompile(`^[0-9]{10}-[a-f0-9]{64}$`)
var verifier = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func ValidClient(value string) bool {
	return len(value) > 0 && len(value) <= 100 && utf8.ValidString(value) && strings.TrimSpace(value) == value && !strings.ContainsFunc(value, unicode.IsControl)
}
func requestID(at time.Time, client, secret string) string {
	timestamp := strconv.FormatInt(at.UTC().Unix(), 10)
	return timestamp + "-" + sessions.Digest(timestamp+"\x00"+client+"\x00"+secret)
}
func NewRequest(at time.Time, client string) (id, secret, code string, err error) {
	if !ValidClient(client) {
		return "", "", "", errors.New("CLI client name must contain 1 through 100 characters without control characters")
	}
	secret = sessions.Random()
	id = requestID(at, client, secret)
	return id, secret, Code(id), nil
}
func Code(id string) string {
	if !challenge.MatchString(id) {
		return ""
	}
	_, hash, _ := strings.Cut(id, "-")
	return strings.ToUpper(hash[:4] + "-" + hash[4:8])
}
func expires(id string, now time.Time) (time.Time, error) {
	if !challenge.MatchString(id) {
		return time.Time{}, ErrInvalid
	}
	timestamp, _, _ := strings.Cut(id, "-")
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, ErrInvalid
	}
	at := time.Unix(seconds, 0).UTC()
	until := at.Add(Lifetime)
	if at.After(now.Add(30*time.Second)) || !now.Before(until) {
		return time.Time{}, ErrInvalid
	}
	return until, nil
}
func verify(proof Proof, now time.Time) error {
	at, err := expires(proof.Request, now)
	if err != nil || !ValidClient(proof.Client) || !verifier.MatchString(proof.Verifier) {
		return ErrInvalid
	}
	expected := requestID(at.Add(-Lifetime), proof.Client, proof.Verifier)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(proof.Request)) != 1 {
		return ErrInvalid
	}
	return nil
}
func terminal(state string) bool {
	return state == "consumed" || state == "denied" || state == "cancelled"
}
func codeMatches(id, code string) bool {
	return subtle.ConstantTimeCompare([]byte(Code(id)), []byte(strings.ToUpper(strings.TrimSpace(code)))) == 1
}
