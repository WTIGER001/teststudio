package statuspages

import (
	"github.com/wtiger001/teststudio/internal/reliability"
	"time"
)

type Entry struct {
	Probe       string `json:"probeId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suite       string `json:"suiteId,omitempty"`
	Test        string `json:"testId,omitempty"`
	Environment string `json:"environmentId,omitempty"`
	Location    string `json:"executionLocation,omitempty"`
	Monitor     bool   `json:"monitor,omitempty"`
}
type Definition struct {
	Logo       string  `json:"logo,omitempty"`
	LogoAlt    string  `json:"logoAlt,omitempty"`
	Slug       string  `json:"slug,omitempty"`
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Background string  `json:"background"`
	Window     string  `json:"defaultWindow"`
	MaxAge     int     `json:"maxAgeSeconds"`
	Probes     []Entry `json:"probes"`
}
type Page struct {
	Definition
	Token       string      `json:"publicToken"`
	Published   *Definition `json:"published,omitempty"`
	Publisher   string      `json:"publisher,omitempty"`
	PublishedAt *time.Time  `json:"publishedAt,omitempty"`
}
type Point struct {
	Created time.Time `json:"-"`
	Hidden  bool      `json:"-"`
	At      time.Time `json:"at"`
	State   string    `json:"state"`
}
type Availability struct {
	Percent  *float64  `json:"percent"`
	Coverage float64   `json:"coveragePercent"`
	Up       float64   `json:"upSeconds"`
	Down     float64   `json:"downSeconds"`
	Unknown  float64   `json:"unknownSeconds"`
	Since    time.Time `json:"since"`
}
type Row struct {
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	State        string               `json:"state"`
	History      []Point              `json:"history"`
	Availability Availability         `json:"availability"`
	Metric       string               `json:"metric"`
	Buckets      []reliability.Bucket `json:"buckets"`
	Totals       reliability.Totals   `json:"totals"`
	PassRate     *float64             `json:"passRate"`
	Contexts     int                  `json:"contextCount"`
}
type Report struct {
	Logo       string    `json:"logo,omitempty"`
	LogoAlt    string    `json:"logoAlt,omitempty"`
	Title      string    `json:"title"`
	Background string    `json:"background"`
	Window     string    `json:"window"`
	MaxAge     int       `json:"maxAgeSeconds"`
	At         time.Time `json:"asOf"`
	Probes     []Row     `json:"probes"`
}
