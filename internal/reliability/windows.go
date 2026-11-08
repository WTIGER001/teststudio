package reliability

import (
	"errors"
	"time"
)

func WindowStart(window string, now time.Time) (time.Time, error) {
	days, ok := map[string]int{"24h": 1, "7d": 7, "1m": 30, "1y": 365, "all": 0}[window]
	if !ok {
		return time.Time{}, errors.New("choose 24h, 7d, 1m, 1y, or all")
	}
	if days == 0 {
		return time.Time{}, nil
	}
	return now.Add(-time.Duration(days) * 24 * time.Hour), nil
}

func Resolution(since, until time.Time) string {
	span := until.Sub(since)
	switch {
	case span <= 24*time.Hour:
		return "hour"
	case span <= 7*24*time.Hour:
		return "6h"
	case span <= 31*24*time.Hour:
		return "day"
	case span <= 366*24*time.Hour:
		return "week"
	case span <= 5*366*24*time.Hour:
		return "month"
	default:
		return "year"
	}
}

func boundary(at time.Time, resolution string) (time.Time, error) {
	at = at.UTC()
	switch resolution {
	case "hour":
		return at.Truncate(time.Hour), nil
	case "6h":
		return at.Truncate(6 * time.Hour), nil
	case "day":
		return at.Truncate(24 * time.Hour), nil
	case "week":
		day := at.Truncate(24 * time.Hour)
		return day.Add(-time.Duration((int(day.Weekday())+6)%7) * 24 * time.Hour), nil
	case "month":
		return time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	case "year":
		return time.Date(at.Year(), 1, 1, 0, 0, 0, 0, time.UTC), nil
	default:
		return time.Time{}, errors.New("choose hour, 6h, day, week, month, or year resolution")
	}
}

func advance(at time.Time, resolution string) time.Time {
	switch resolution {
	case "hour":
		return at.Add(time.Hour)
	case "6h":
		return at.Add(6 * time.Hour)
	case "day":
		return at.Add(24 * time.Hour)
	case "week":
		return at.Add(7 * 24 * time.Hour)
	case "month":
		return at.AddDate(0, 1, 0)
	default:
		return at.AddDate(1, 0, 0)
	}
}
