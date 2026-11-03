package passwords

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

const prefix = "$argon2id$v=19$m=65536,t=3,p=4$"

var ErrBusy = errors.New("password processing is busy; try again shortly")
var workers = make(chan struct{}, 2)
var common = map[string]bool{"passwordpassword": true, "password123456789": true, "123456789012345": true, "qwertyuiopasdfgh": true, "teststudiopassword": true, "letmeinletmeinletmein": true, "correcthorsebatterystaple": true}

func Normalize(value string) string { return norm.NFC.String(value) }

func Validate(value string) error {
	if !utf8.ValidString(value) || len(value) > 1024 {
		return errors.New("use a password of 15 through 256 characters and at most 1024 bytes")
	}
	value = Normalize(value)
	n := utf8.RuneCountInString(value)
	if !utf8.ValidString(value) || n < 15 || n > 256 || len(value) > 1024 {
		return errors.New("use a password of 15 through 256 characters and at most 1024 bytes")
	}
	same, first := true, rune(0)
	for i, r := range value {
		if unicode.IsControl(r) {
			return errors.New("passwords cannot contain control characters")
		}
		if i == 0 {
			first = r
		} else if r != first {
			same = false
		}
	}
	if same || common[strings.ToLower(value)] {
		return errors.New("choose a password that is not a common or repeated value")
	}
	return nil
}

func derive(ctx context.Context, value string, salt []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case workers <- struct{}{}:
		defer func() { <-workers }()
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, ErrBusy
	}
	key := argon2.IDKey([]byte(Normalize(value)), salt, 3, 64*1024, 4, 32)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return key, nil
}

func Hash(ctx context.Context, value string) (string, error) {
	if err := Validate(value); err != nil {
		return "", err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := derive(ctx, value, salt)
	if err != nil {
		return "", err
	}
	return prefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

func parse(encoded string) ([]byte, []byte, bool) {
	if !strings.HasPrefix(encoded, prefix) {
		return nil, nil, false
	}
	parts := strings.Split(strings.TrimPrefix(encoded, prefix), "$")
	if len(parts) != 2 {
		return nil, nil, false
	}
	salt, a := base64.RawStdEncoding.Strict().DecodeString(parts[0])
	key, b := base64.RawStdEncoding.Strict().DecodeString(parts[1])
	return salt, key, a == nil && b == nil && len(salt) == 16 && len(key) == 32
}

func ValidHash(encoded string) bool { _, _, ok := parse(encoded); return ok }

func Verify(ctx context.Context, encoded, value string) (bool, error) {
	salt, key, ok := parse(encoded)
	if !ok || !utf8.ValidString(value) || len(value) > 1024 {
		return false, nil
	}
	actual, err := derive(ctx, value, salt)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(key, actual) == 1, nil
}

func DummyVerify(ctx context.Context, value string) error {
	_, err := derive(ctx, value, []byte("TestStudioDummy!"))
	return err
}
