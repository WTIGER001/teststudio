package timing

import "time"

type Context struct {
	Project     string `json:"projectId"`
	Suite       string `json:"suiteId"`
	Test        string `json:"testId"`
	Environment string `json:"environmentId,omitempty"`
	Location    string `json:"executionLocation"`
	Runner      string `json:"runner"`
	Profile     string `json:"profile"`
	Definition  string `json:"definitionFingerprint"`
	Target      string `json:"targetFingerprint"`
}

type Sample struct {
	Run        string    `json:"runId"`
	DurationMs float64   `json:"durationMs"`
	At         time.Time `json:"at"`
}

type Window struct {
	Context     Context    `json:"context"`
	Generation  int64      `json:"generation"`
	Samples     []Sample   `json:"samples"`
	ResetAt     *time.Time `json:"resetAt,omitempty"`
	ResetReason string     `json:"resetReason,omitempty"`
}

type Prediction struct {
	Key                string     `json:"baselineId"`
	Context            Context    `json:"context"`
	Generation         int64      `json:"generation"`
	Revision           int64      `json:"baselineRevision"`
	Policy             Policy     `json:"policy"`
	Samples            int        `json:"sampleCount"`
	MedianMs           *float64   `json:"expectedMs,omitempty"`
	P95Ms              *float64   `json:"p95Ms,omitempty"`
	ThresholdMs        *float64   `json:"slowThresholdMs,omitempty"`
	SuggestedTimeoutMs *int       `json:"suggestedTimeoutMs,omitempty"`
	EffectiveTimeoutMs int        `json:"effectiveTimeoutMs"`
	Ready              bool       `json:"ready"`
	FirstSampleAt      *time.Time `json:"firstSampleAt,omitempty"`
	LastSampleAt       *time.Time `json:"lastSampleAt,omitempty"`
	AsOf               time.Time  `json:"asOf"`
}

type Assessment struct {
	Prediction Prediction `json:"baseline"`
	Status     string     `json:"status"`
	Ratio      *float64   `json:"ratio,omitempty"`
	Censored   bool       `json:"censored"`
	TimeoutMs  int        `json:"timeoutMs"`
}

type Estimate struct {
	Tests           map[string]Prediction `json:"tests"`
	ExpectedMs      *float64              `json:"expectedExecutionMs,omitempty"`
	P95Ms           *float64              `json:"highExecutionMs,omitempty"`
	ReadyTests      int                   `json:"readyTests"`
	MeasuredTests   int                   `json:"measuredTests"`
	TotalTests      int                   `json:"totalTests"`
	Provisional     bool                  `json:"provisional"`
	TimeoutBudgetMs int64                 `json:"testTimeoutBudgetMs"`
	ExceedsRunLimit bool                  `json:"exceedsRunLimit"`
}
