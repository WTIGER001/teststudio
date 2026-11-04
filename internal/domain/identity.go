package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/wtiger001/teststudio/internal/identifier"
)

func ValidID(value string) bool { return identifier.Valid(value) }

func NewID() string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return time.Now().UTC().Format("20060102T150405.000000000") + "-" + hex.EncodeToString(bytes[:])
}
