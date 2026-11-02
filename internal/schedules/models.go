package schedules

import (
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/timing"
	"time"
)

type Definition struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Enabled         bool         `json:"enabled"`
	Kind            string       `json:"kind"`
	IntervalSeconds int          `json:"intervalSeconds,omitempty"`
	Cron            string       `json:"cron,omitempty"`
	Timezone        string       `json:"timezone"`
	Overlap         string       `json:"overlap"`
	Request         runs.Request `json:"request"`
}
type Schedule struct {
	Definition
	Project    string     `json:"projectId"`
	Owner      string     `json:"owner"`
	Generation int64      `json:"generation"`
	Archived   bool       `json:"archived"`
	NextAt     *time.Time `json:"nextAt,omitempty"`
	LastAt     *time.Time `json:"lastAt,omitempty"`
	LastRun    string     `json:"lastRunId,omitempty"`
	LastState  string     `json:"lastState,omitempty"`
	LastError  string     `json:"lastError,omitempty"`
	Claim      string     `json:"claim,omitempty"`
	ClaimUntil *time.Time `json:"claimUntil,omitempty"`
}
type Firing struct {
	ID         string    `json:"id"`
	Schedule   string    `json:"scheduleId"`
	Generation int64     `json:"generation"`
	Owner      string    `json:"owner"`
	DueAt      time.Time `json:"dueAt"`
	At         time.Time `json:"at"`
	State      string    `json:"state"`
	Run        string    `json:"runId,omitempty"`
	Error      string    `json:"error,omitempty"`
	Coalesced  bool      `json:"coalesced"`
}
type Preview struct {
	Next   []time.Time     `json:"next"`
	Timing timing.Estimate `json:"timing"`
}
type work struct{ Project, Schedule string }
