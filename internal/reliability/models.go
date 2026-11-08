package reliability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/wtiger001/teststudio/internal/timing"
	"time"
)

type Scope struct {
	Project     string `json:"projectId"`
	Suite       string `json:"suiteId"`
	Test        string `json:"testId"`
	Environment string `json:"environmentId,omitempty"`
	Location    string `json:"executionLocation"`
}

func Hash(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s Scope) ID() string { return Hash(s) }

type Observation struct {
	Scope
	Run           string             `json:"runId"`
	Title         string             `json:"title"`
	Type          string             `json:"type"`
	Runner        string             `json:"runner"`
	Context       string             `json:"context"`
	Match         string             `json:"match"`
	Created       time.Time          `json:"createdAt"`
	At            time.Time          `json:"at"`
	State         string             `json:"state"`
	ObservedState string             `json:"observedState,omitempty"`
	FailureKind   string             `json:"failureKind,omitempty"`
	DurationMs    *float64           `json:"durationMs"`
	Censored      bool               `json:"censored"`
	Started       time.Time          `json:"startedAt"`
	Timing        *timing.Assessment `json:"timing,omitempty"`
	MonitorState  string             `json:"monitorState"`
	Source        json.RawMessage    `json:"source"`
}

type Counts struct {
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	TimedOut int `json:"timedOut"`
	Flaky    int `json:"flaky"`
	Skipped  int `json:"skipped"`
	Unknown  int `json:"unknown"`
}

func (c Counts) Total() int {
	return c.Passed + c.Failed + c.TimedOut + c.Flaky + c.Skipped + c.Unknown
}
func (c Counts) Known() int { return c.Passed + c.Failed + c.TimedOut + c.Flaky }

type Totals struct {
	Counts        Counts  `json:"counts"`
	Up            float64 `json:"upSeconds"`
	Down          float64 `json:"downSeconds"`
	Unknown       float64 `json:"unknownSeconds"`
	DurationSum   float64 `json:"durationSumMs"`
	DurationCount int     `json:"durationSampleCount"`
	Censored      int     `json:"censoredCount"`
	Unavailable   int     `json:"durationUnavailableCount"`
}

func (t *Totals) Add(v Totals) {
	t.Counts.Passed += v.Counts.Passed
	t.Counts.Failed += v.Counts.Failed
	t.Counts.TimedOut += v.Counts.TimedOut
	t.Counts.Flaky += v.Counts.Flaky
	t.Counts.Skipped += v.Counts.Skipped
	t.Counts.Unknown += v.Counts.Unknown
	t.Up += v.Up
	t.Down += v.Down
	t.DurationSum += v.DurationSum
	t.DurationCount += v.DurationCount
	t.Censored += v.Censored
	t.Unavailable += v.Unavailable
}

func (t *Totals) Observe(o Observation) {
	switch o.State {
	case "passed":
		t.Counts.Passed++
	case "failed":
		t.Counts.Failed++
	case "timed-out":
		t.Counts.TimedOut++
	case "flaky":
		t.Counts.Flaky++
	case "skipped":
		t.Counts.Skipped++
	default:
		t.Counts.Unknown++
	}
	if o.DurationMs == nil {
		t.Unavailable++
	} else if o.Censored {
		t.Censored++
	} else {
		t.DurationSum += *o.DurationMs
		t.DurationCount++
	}
}

type Part struct {
	Run     string          `json:"runId"`
	First   time.Time       `json:"firstAt"`
	Source  json.RawMessage `json:"source"`
	Context string          `json:"context"`
	Match   string          `json:"match"`
	Totals  Totals          `json:"totals"`
}

type Summary struct {
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
	Parts []Part    `json:"parts"`
}

type Availability struct {
	Percent  *float64  `json:"percent"`
	Coverage float64   `json:"coveragePercent"`
	Up       float64   `json:"upSeconds"`
	Down     float64   `json:"downSeconds"`
	Unknown  float64   `json:"unknownSeconds"`
	Since    time.Time `json:"since"`
}

type Bucket struct {
	Since        time.Time     `json:"since"`
	Until        time.Time     `json:"until"`
	State        string        `json:"state"`
	Partial      bool          `json:"partial"`
	Totals       Totals        `json:"totals"`
	PassRate     *float64      `json:"passRate"`
	Availability *Availability `json:"availability"`
}

type Point struct {
	At    time.Time `json:"at"`
	State string    `json:"state"`
}

type Row struct {
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	State        string       `json:"state"`
	Metric       string       `json:"metric"`
	History      []Point      `json:"history"`
	Buckets      []Bucket     `json:"buckets"`
	Totals       Totals       `json:"totals"`
	PassRate     *float64     `json:"passRate"`
	Availability Availability `json:"availability"`
	Contexts     int          `json:"contextCount"`
}

type Report struct {
	Title      string    `json:"title"`
	Background string    `json:"background"`
	Window     string    `json:"window"`
	MaxAge     int       `json:"maxAgeSeconds"`
	At         time.Time `json:"asOf"`
	Probes     []Row     `json:"probes"`
}

type Query struct {
	Scope
	Since      time.Time
	Until      time.Time
	Resolution string
	MaxAge     int
	Match      string
}

func ratio(up, total float64) *float64 {
	if total <= 0 {
		return nil
	}
	value := 100 * up / total
	return &value
}
