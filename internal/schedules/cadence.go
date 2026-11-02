package schedules

import (
	"errors"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/robfig/cron/v3"
	"github.com/wtiger001/teststudio/internal/domain"
)

func normalize(d Definition, location string) Definition {
	d.Name = strings.TrimSpace(d.Name)
	d.Cron = strings.Join(strings.Fields(d.Cron), " ")
	if d.Timezone == "" {
		d.Timezone = "UTC"
	}
	if d.Overlap == "" {
		d.Overlap = "skip"
	}
	if d.Request.Location == "" {
		d.Request.Location = location
	}
	if d.Request.TimeoutSeconds == 0 {
		d.Request.TimeoutSeconds = 1800
	}
	return d
}
func validate(d Definition) error {
	if !domain.ValidID(d.ID) || d.Name == "" || len(d.Name) > 256 || !domain.ValidID(d.Request.Suite) || len(d.Request.IDs) > 1000 || d.Request.Parent != "" || d.Request.TimeoutSeconds < 1 || d.Request.TimeoutSeconds > 3600 || d.Request.Retries < 0 || d.Request.Retries > 2 {
		return errors.New("schedule requires valid identifiers, a name, a suite, and bounded run settings")
	}
	seen := map[string]bool{}
	for _, id := range []string{d.Request.Environment, d.Request.Release, d.Request.Location} {
		if id != "" && !domain.ValidID(id) {
			return errors.New("scheduled run references must use valid identifiers")
		}
	}
	for _, id := range d.Request.IDs {
		if !domain.ValidID(id) || seen[id] {
			return errors.New("scheduled test identifiers must be valid and unique")
		}
		seen[id] = true
	}
	if d.Overlap != "skip" && d.Overlap != "allow" {
		return errors.New("overlap policy must be skip or allow")
	}
	if len(d.Timezone) > 128 || d.Timezone == "Local" {
		return errors.New("schedule requires an explicit IANA timezone")
	}
	if _, err := time.LoadLocation(d.Timezone); err != nil {
		return errors.New("schedule timezone is unavailable")
	}
	if d.Kind == "interval" {
		if d.IntervalSeconds < 5 || d.IntervalSeconds > 2592000 || d.Cron != "" || d.Timezone != "UTC" {
			return errors.New("interval schedules require 5 through 2592000 seconds, UTC, and no cron expression")
		}
	} else if d.Kind == "cron" {
		if d.IntervalSeconds != 0 || len(strings.Fields(d.Cron)) != 5 || len(d.Cron) > 256 {
			return errors.New("cron schedules require five fields and no interval")
		}
		if _, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse("CRON_TZ=" + d.Timezone + " " + d.Cron); err != nil {
			return errors.New("cron expression is invalid")
		}
	} else {
		return errors.New("schedule kind must be interval or cron")
	}
	return nil
}

func Next(d Definition, after time.Time) (time.Time, error) {
	if err := validate(d); err != nil {
		return time.Time{}, err
	}
	if d.Kind == "interval" {
		return after.Add(time.Duration(d.IntervalSeconds) * time.Second).UTC(), nil
	}
	spec, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse("CRON_TZ=" + d.Timezone + " " + d.Cron)
	if err != nil {
		return time.Time{}, err
	}
	next := spec.Next(after).UTC()
	if next.IsZero() {
		return next, errors.New("cron expression has no execution within five years")
	}
	return next, nil
}

func advance(d Definition, due, now time.Time) (time.Time, bool, error) {
	first, err := Next(d, due)
	if err != nil {
		return first, false, err
	}
	coalesced := !first.After(now)
	if d.Kind == "interval" {
		interval := time.Duration(d.IntervalSeconds) * time.Second
		return due.Add((now.Sub(due)/interval + 1) * interval).UTC(), coalesced, nil
	}
	next, err := Next(d, now)
	return next, coalesced, err
}
