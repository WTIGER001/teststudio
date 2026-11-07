package statuspages

import (
	"errors"
	"sort"
	"time"
)

func WindowStart(window string, now time.Time) (time.Time, error) {
	days := map[string]int{"24h": 1, "7d": 7, "1m": 30, "1y": 365, "all": 0}
	d, ok := days[window]
	if !ok {
		return time.Time{}, errors.New("choose 24h, 7d, 1m, 1y, or all")
	}
	if d == 0 {
		return time.Time{}, nil
	}
	return now.Add(-time.Duration(d) * 24 * time.Hour), nil
}
func Calculate(points []Point, since, now time.Time, maxAge time.Duration) Availability {
	values := append([]Point{}, points...)
	sort.SliceStable(values, func(i, j int) bool { return values[i].At.Before(values[j].At) })
	if since.IsZero() {
		since = now
		for _, p := range values {
			if p.At.Before(now) {
				since = p.At
				break
			}
		}
	}
	total := max(now.Sub(since).Seconds(), 0)
	result := Availability{Since: since, Unknown: total}
	for i, p := range values {
		start := p.At
		if start.Before(since) {
			start = since
		}
		end := p.At.Add(maxAge)
		if i+1 < len(values) && values[i+1].At.Before(end) {
			end = values[i+1].At
		}
		if end.After(now) {
			end = now
		}
		if !end.After(start) {
			continue
		}
		seconds := end.Sub(start).Seconds()
		switch p.State {
		case "passed":
			result.Up += seconds
		case "failed", "timed-out", "flaky":
			result.Down += seconds
		}
	}
	known := result.Up + result.Down
	result.Unknown = max(0, total-known)
	if total > 0 {
		result.Coverage = 100 * known / total
	}
	if known > 0 {
		percentage := 100 * result.Up / known
		result.Percent = &percentage
	}
	return result
}
