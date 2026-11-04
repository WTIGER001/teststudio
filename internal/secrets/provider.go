package secrets

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/networkguard"
)

const NotificationWebhook = "notification-webhook"
const APICredential = "api-credential"

var ErrUnavailable = errors.New("secret reference is unavailable")

type Reference struct {
	Origins []string `json:"allowedOrigins,omitempty"`
	ID      string   `json:"id"`
	Name    string   `json:"name"`
}
type Value struct {
	Origins []string           `json:"-"`
	Bytes   []byte             `json:"-"`
	Network networkguard.Scope `json:"-"`
}
type Provider interface {
	References(context.Context, string, string) ([]Reference, error)
	Resolve(context.Context, string, string, string) (Value, error)
}
