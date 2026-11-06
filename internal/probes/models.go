package probes

import (
	"github.com/wtiger001/teststudio/internal/execution"
	"time"
)

type Binding struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Suite       string `json:"suiteId"`
	Test        string `json:"testId"`
	Environment string `json:"environmentId"`
	Endpoint    string `json:"endpoint"`
	Location    string `json:"executionLocation"`
	Archived    bool   `json:"archived"`
}
type Observation struct {
	Run           string                     `json:"runId"`
	Release       string                     `json:"releaseId"`
	CreatedAt     time.Time                  `json:"createdAt"`
	At            *time.Time                 `json:"observedAt,omitempty"`
	State         string                     `json:"state"`
	Error         string                     `json:"error,omitempty"`
	Evidence      string                     `json:"evidenceState"`
	HTTP          *execution.HTTPObservation `json:"http,omitempty"`
	TargetChanged bool                       `json:"targetChanged"`
}
type Schedule struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Enabled   bool       `json:"enabled"`
	Kind      string     `json:"kind"`
	Interval  int        `json:"intervalSeconds,omitempty"`
	Cron      string     `json:"cron,omitempty"`
	Next      *time.Time `json:"nextAt,omitempty"`
	LastState string     `json:"lastState,omitempty"`
	Shared    bool       `json:"shared"`
}
type Summary struct {
	Binding
	Revision      int64         `json:"revision"`
	Application   string        `json:"applicationId,omitempty"`
	Stage         string        `json:"stage,omitempty"`
	URL           string        `json:"url,omitempty"`
	State         string        `json:"state"`
	Freshness     string        `json:"freshness"`
	Availability  string        `json:"availability"`
	Latest        *Observation  `json:"latest,omitempty"`
	Active        *Observation  `json:"active,omitempty"`
	History       []Observation `json:"history"`
	Schedules     []Schedule    `json:"schedules"`
	ScheduleState string        `json:"scheduleState"`
	Samples       int           `json:"samples"`
	Passed        int           `json:"passed"`
	Other         int           `json:"other"`
	Rate          *float64      `json:"successRate,omitempty"`
	Partial       bool          `json:"partial"`
	ActiveRelease string        `json:"activeReleaseId,omitempty"`
}
type Query struct {
	Search      string `json:"search"`
	Application string `json:"application"`
	Stage       string `json:"stage"`
	State       string `json:"state"`
	Freshness   string `json:"freshness"`
	Schedule    string `json:"schedule"`
	Archived    bool   `json:"archived"`
	Limit       int    `json:"limit"`
	Hours       int    `json:"hours"`
}
type Page struct {
	Items   []Summary `json:"items"`
	Matched int       `json:"matched"`
	Cursor  string    `json:"cursor,omitempty"`
	Now     time.Time `json:"serverTime"`
	Since   time.Time `json:"windowStart"`
	Partial bool      `json:"partial"`
	Scanned int       `json:"scanned"`
}
