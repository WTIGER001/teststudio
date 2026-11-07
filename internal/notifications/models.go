package notifications

import (
	"context"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/secrets"
	"time"
)

type Definition struct {
	Application  string   `json:"applicationId,omitempty"`
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	Kind         string   `json:"kind"`
	Reference    string   `json:"secretReference"`
	Suites       []string `json:"suiteIds"`
	Environments []string `json:"environmentIds"`
	Stages       []string `json:"stages"`
	Triggers     []string `json:"triggers"`
	Outcomes     []string `json:"outcomes"`
	EvidenceGaps bool     `json:"includeEvidenceGaps"`
	MaxAttempts  int      `json:"maxAttempts"`
}
type Channel struct {
	Definition
	Project  string `json:"projectId"`
	Owner    string `json:"owner"`
	Archived bool   `json:"archived"`
	Epoch    int64  `json:"deliveryEpoch"`
}
type subscription struct {
	Channel  Channel `json:"channel"`
	Revision int64   `json:"revision"`
}
type plan struct {
	Project  string         `json:"projectId"`
	Run      string         `json:"runId"`
	Channels []subscription `json:"channels"`
}
type registry struct {
	IDs []string `json:"ids"`
}
type Counts = runs.Counts
type Message struct {
	Schema      string    `json:"schemaVersion"`
	Event       string    `json:"event"`
	Delivery    string    `json:"deliveryId"`
	Project     string    `json:"projectId"`
	Run         string    `json:"runId"`
	Suite       string    `json:"suiteId"`
	Environment string    `json:"environmentId,omitempty"`
	Stage       string    `json:"stage,omitempty"`
	Trigger     string    `json:"trigger"`
	State       string    `json:"state"`
	Outcome     string    `json:"outcome"`
	Evidence    string    `json:"evidenceState"`
	Counts      Counts    `json:"counts"`
	FinishedAt  time.Time `json:"finishedAt"`
	URL         string    `json:"url"`
}
type Delivery struct {
	ID                string       `json:"id"`
	Project           string       `json:"projectId"`
	Run               string       `json:"runId"`
	Channel           string       `json:"channelId"`
	ChannelName       string       `json:"channelName"`
	DispatchEpoch     int64        `json:"deliveryEpoch"`
	Destination       subscription `json:"destination"`
	Message           Message      `json:"message"`
	State             string       `json:"state"`
	Attempts          int          `json:"attempts"`
	AttemptLimit      int          `json:"attemptLimit"`
	NextAt            time.Time    `json:"nextAt"`
	HTTPStatus        int          `json:"httpStatus,omitempty"`
	Error             string       `json:"error,omitempty"`
	AcceptanceUnknown bool         `json:"acceptanceUnknown"`
	Claim             string       `json:"claimToken,omitempty"`
	ClaimUntil        *time.Time   `json:"claimUntil,omitempty"`
	CreatedAt         time.Time    `json:"createdAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
	DeliveredAt       *time.Time   `json:"deliveredAt,omitempty"`
}
type Attempt struct {
	Number            int        `json:"number"`
	State             string     `json:"state"`
	StartedAt         time.Time  `json:"startedAt"`
	FinishedAt        *time.Time `json:"finishedAt,omitempty"`
	HTTPStatus        int        `json:"httpStatus,omitempty"`
	Error             string     `json:"error,omitempty"`
	AcceptanceUnknown bool       `json:"acceptanceUnknown"`
}
type Outcome struct {
	HTTPStatus        int
	Retryable         bool
	RetryAfter        time.Duration
	Error             string
	AcceptanceUnknown bool
	GlobalRateLimit   bool
}
type Sender interface {
	Send(context.Context, string, secrets.Value, Message) Outcome
}
type job struct {
	Project  string `json:"projectId"`
	Delivery string `json:"deliveryId"`
}
