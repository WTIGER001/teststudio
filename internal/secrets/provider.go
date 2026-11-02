package secrets

import (
	"context"
	"errors"
	"github.com/wtiger001/teststudio/internal/networkguard"
)

const NotificationWebhook = "notification-webhook"

var ErrUnavailable = errors.New("secret reference is unavailable")

type Reference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Value struct {
	Bytes   []byte             `json:"-"`
	Network networkguard.Scope `json:"-"`
}
type Provider interface {
	References(context.Context, string, string) ([]Reference, error)
	Resolve(context.Context, string, string, string) (Value, error)
}
