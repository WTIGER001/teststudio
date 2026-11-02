package capture

import (
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
)

type Policy struct {
	Logs                 bool           `json:"logs"`
	Screenshots          string         `json:"screenshots"`
	Trace                string         `json:"trace"`
	Video                string         `json:"video"`
	MaxBytes             int64          `json:"maxBytes,omitempty"`
	MaxFileBytes         int64          `json:"maxFileBytes,omitempty"`
	RetentionDays        int            `json:"retentionDays,omitempty"`
	KindRetention        map[string]int `json:"kindRetentionDays,omitempty"`
	FailureRetentionDays int            `json:"failureRetentionDays,omitempty"`
}

type Limits struct {
	ProjectBytes  int64 `json:"projectBytes"`
	RunBytes      int64 `json:"runBytes"`
	FileBytes     int64 `json:"fileBytes"`
	RetentionDays int   `json:"retentionDays"`
}

func DefaultPolicy() Policy {
	return Policy{Logs: true, Screenshots: "failure", Trace: "failure", Video: "off", MaxBytes: 100 << 20, MaxFileBytes: 64 << 20, RetentionDays: 30}
}
func DefaultLimits() Limits {
	return Limits{ProjectBytes: 10 << 30, RunBytes: 1 << 30, FileBytes: 64 << 20, RetentionDays: 3650}
}

func Normalize(p Policy) Policy {
	if !p.Logs && p.Screenshots == "" && p.Trace == "" && p.Video == "" && p.MaxBytes == 0 && p.MaxFileBytes == 0 && p.RetentionDays == 0 && len(p.KindRetention) == 0 && p.FailureRetentionDays == 0 {
		return DefaultPolicy()
	}
	if p.MaxBytes == 0 {
		p.MaxBytes = 100 << 20
	}
	if p.MaxFileBytes == 0 {
		p.MaxFileBytes = min(64<<20, p.MaxBytes)
	}
	if p.RetentionDays == 0 {
		p.RetentionDays = 30
	}
	if len(p.KindRetention) == 0 {
		p.KindRetention = nil
	} else {
		p.KindRetention = maps.Clone(p.KindRetention)
	}
	return p
}

func Resolve(project, suite *Policy) Policy {
	p := DefaultPolicy()
	if project != nil {
		p = Normalize(*project)
	}
	if suite != nil {
		p = Normalize(*suite)
	}
	return p
}

func ResolveLimits(limits *Limits) Limits {
	if limits == nil {
		return DefaultLimits()
	}
	return *limits
}

func ValidateLimits(l Limits) error {
	if l.ProjectBytes < 1 || l.ProjectBytes > 1<<50 || l.RunBytes < 1 || l.RunBytes > 1<<30 || l.FileBytes < 1 || l.FileBytes > 64<<20 || l.RetentionDays < 1 || l.RetentionDays > 3650 {
		return errors.New("capture limits require positive quotas, up to 1 GiB per run, 64 MiB per file, and 3650 retention days")
	}
	return nil
}

func Validate(p Policy, l Limits) error {
	if err := ValidateLimits(l); err != nil {
		return err
	}
	p = Normalize(p)
	for _, mode := range []string{p.Screenshots, p.Trace, p.Video} {
		if !slices.Contains([]string{"off", "failure", "always"}, mode) {
			return errors.New("capture modes must be off, failure, or always")
		}
	}
	if p.MaxBytes < 1 || p.MaxBytes > l.RunBytes || p.MaxFileBytes < 1 || p.MaxFileBytes > l.FileBytes || p.MaxFileBytes > p.MaxBytes {
		return errors.New("capture byte limits exceed the project or run limit")
	}
	if p.RetentionDays < 1 || p.RetentionDays > l.RetentionDays || p.FailureRetentionDays < 0 || p.FailureRetentionDays > l.RetentionDays {
		return errors.New("capture retention exceeds the project limit")
	}
	if len(p.KindRetention) > 32 {
		return errors.New("capture supports up to 32 artifact retention rules")
	}
	for kind, days := range p.KindRetention {
		if !domain.ValidID(kind) || days < 1 || days > l.RetentionDays {
			return errors.New("artifact retention rules require a valid kind and permitted duration")
		}
	}
	return nil
}

func Expiry(created time.Time, p Policy, kind, outcome string) time.Time {
	p = Normalize(p)
	days := p.RetentionDays
	if value, ok := p.KindRetention[kind]; ok {
		days = value
	}
	if outcome == "failed" || outcome == "flaky" {
		days = max(days, p.FailureRetentionDays)
	}
	return created.Add(time.Duration(days) * 24 * time.Hour)
}
