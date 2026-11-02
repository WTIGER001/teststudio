package trends

import "time"

type Filter struct {
	Environment string    `json:"environment,omitempty"`
	Suite       string    `json:"suite,omitempty"`
	Category    string    `json:"category,omitempty"`
	Test        string    `json:"test,omitempty"`
	Location    string    `json:"location,omitempty"`
	Since       time.Time `json:"since"`
	Until       time.Time `json:"until"`
}

type Counts struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Failed   int `json:"failed"`
	TimedOut int `json:"timedOut"`
	Flaky    int `json:"flaky"`
	Skipped  int `json:"skipped"`
	Unknown  int `json:"unknown"`
}

type Point struct {
	Run         string    `json:"runId"`
	At          time.Time `json:"at"`
	Suite       string    `json:"suiteId"`
	Environment string    `json:"environmentId,omitempty"`
	Location    string    `json:"executionLocation"`
	State       string    `json:"state"`
	Outcome     string    `json:"outcome"`
	Counts      Counts    `json:"counts"`
	Gap         string    `json:"gap,omitempty"`
}

type Ranking struct {
	Suite       string   `json:"suiteId"`
	Test        string   `json:"testId"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Counts      Counts   `json:"counts"`
	Comparisons int      `json:"comparisons"`
	Changes     int      `json:"changes"`
	Volatility  *float64 `json:"volatility,omitempty"`
}

type Option struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Options struct {
	Suites       []Option `json:"suites"`
	Environments []Option `json:"environments"`
	Categories   []string `json:"categories"`
	Locations    []string `json:"locations"`
	Partial      bool     `json:"partial"`
}

type Report struct {
	Project            string    `json:"projectId"`
	Filter             Filter    `json:"filter"`
	Points             []Point   `json:"points"`
	Totals             Counts    `json:"totals"`
	Samples            int       `json:"sampleCount"`
	ContextRuns        int       `json:"contextRuns"`
	FailureRankings    []Ranking `json:"failureRankings"`
	VolatilityRankings []Ranking `json:"volatilityRankings"`
	Options            Options   `json:"options"`
	Cursor             string    `json:"cursor,omitempty"`
	Partial            bool      `json:"partial"`
}
