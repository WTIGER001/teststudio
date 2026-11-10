package projects

import (
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/timing"
)

type Project struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Archived      bool            `json:"archived"`
	Timing        *timing.Policy  `json:"timing,omitempty"`
	Capture       *capture.Policy `json:"capture,omitempty"`
	CaptureLimits *capture.Limits `json:"captureLimits,omitempty"`
}

type Application struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Requirements []Requirement `json:"requirements,omitempty"`
	DefaultSuite string        `json:"defaultSuiteId,omitempty"`
}

type Environment struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Application  string            `json:"applicationId"`
	Stage        string            `json:"stage"`
	Protected    bool              `json:"protected"`
	Disabled     bool              `json:"disabled"`
	AllowHTTP    bool              `json:"allowHttp,omitempty"`
	AllowPrivate bool              `json:"allowPrivateNetwork,omitempty"`
	Endpoints    map[string]string `json:"endpoints"`
	Parameters   map[string]string `json:"parameters,omitempty"`
	Credentials  map[string]string `json:"credentialBindings,omitempty"`
	Locations    []string          `json:"executionLocations,omitempty"`
}

type Suite struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Applications   []string        `json:"applications"`
	RequiresTarget bool            `json:"requiresTarget"`
	ActiveRelease  string          `json:"activeReleaseId,omitempty"`
	Archived       bool            `json:"archived"`
	Timing         *timing.Policy  `json:"timing,omitempty"`
	Capture        *capture.Policy `json:"capture,omitempty"`
}
