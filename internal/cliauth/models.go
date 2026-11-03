package cliauth

import "time"

const Lifetime = 5 * time.Minute
const PollInterval = 2 * time.Second

type Confirmation struct {
	Client  string `json:"clientName"`
	Code    string `json:"code"`
	Approve bool   `json:"approve"`
}
type Proof struct {
	Request  string `json:"requestId"`
	Client   string `json:"clientName"`
	Verifier string `json:"verifier"`
}
type View struct {
	ID        string    `json:"id"`
	Client    string    `json:"clientName"`
	State     string    `json:"state"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type Grant struct {
	Cookie    string    `json:"cookie"`
	CSRF      string    `json:"csrfToken"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type Exchange struct {
	State   string `json:"state"`
	Session *Grant `json:"session,omitempty"`
}
type Options struct {
	ServerTime      time.Time `json:"serverTime"`
	LifetimeSeconds int       `json:"lifetimeSeconds"`
	PollSeconds     int       `json:"pollSeconds"`
}
type approval struct {
	View
	Parent  string `json:"parentSession"`
	Session string `json:"issuedSession,omitempty"`
}
