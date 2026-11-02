package timing

import (
	"errors"
	"math"
	"slices"
)

type Policy struct {
	Mode              string  `json:"mode"`
	SlowdownFactor    float64 `json:"slowdownFactor"`
	MinimumIncreaseMs float64 `json:"minimumIncreaseMs"`
	MinimumSamples    int     `json:"minimumSamples"`
	Window            int     `json:"window"`
	MaxAgeDays        int     `json:"maxAgeDays"`
	TimeoutMode       string  `json:"timeoutMode"`
	TimeoutFactor     float64 `json:"timeoutFactor"`
	TimeoutMarginMs   float64 `json:"timeoutMarginMs"`
	MinimumTimeoutMs  int     `json:"minimumTimeoutMs"`
}

func DefaultPolicy() Policy {
	return Policy{Mode: "fail", SlowdownFactor: 2, MinimumIncreaseMs: 100, MinimumSamples: 5, Window: 50, MaxAgeDays: 30, TimeoutMode: "adaptive", TimeoutFactor: 3, TimeoutMarginMs: 250, MinimumTimeoutMs: 1000}
}
func Normalize(p Policy) Policy {
	if p == (Policy{}) {
		return DefaultPolicy()
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
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func Validate(p Policy) error {
	p = Normalize(p)
	if !slices.Contains([]string{"off", "warn", "fail"}, p.Mode) || !slices.Contains([]string{"fixed", "adaptive"}, p.TimeoutMode) {
		return errors.New("timing requires off, warn, or fail mode and fixed or adaptive timeouts")
	}
	if !finite(p.SlowdownFactor) || p.SlowdownFactor < 1 || p.SlowdownFactor > 100 || !finite(p.MinimumIncreaseMs) || p.MinimumIncreaseMs < 0 || p.MinimumIncreaseMs > 3600000 {
		return errors.New("timing slowdown factor or minimum increase is invalid")
	}
	if p.MinimumSamples < 3 || p.MinimumSamples > 1000 || p.Window < p.MinimumSamples || p.Window > 1000 || p.MaxAgeDays < 1 || p.MaxAgeDays > 3650 {
		return errors.New("timing requires 3 through 1000 samples within a valid window and age")
	}
	if !finite(p.TimeoutFactor) || p.TimeoutFactor < 1 || p.TimeoutFactor > 100 || !finite(p.TimeoutMarginMs) || p.TimeoutMarginMs < 0 || p.TimeoutMarginMs > 3600000 || p.MinimumTimeoutMs < 1 || p.MinimumTimeoutMs > 3600000 {
		return errors.New("adaptive timeout bounds are invalid")
	}
	return nil
}
