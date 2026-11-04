package runs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/paging"
	"github.com/wtiger001/teststudio/internal/storage"
)

type TestHistoryItem struct {
	Run           string     `json:"runId"`
	Release       string     `json:"releaseId"`
	Title         string     `json:"title"`
	CreatedAt     time.Time  `json:"createdAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	Environment   string     `json:"environmentName,omitempty"`
	Location      string     `json:"executionLocation"`
	State         string     `json:"runState"`
	Outcome       string     `json:"runOutcome"`
	TestState     string     `json:"testState"`
	Attempt       int        `json:"attempt"`
	Duration      *float64   `json:"durationMs,omitempty"`
	Error         string     `json:"error,omitempty"`
	EvidenceState string     `json:"evidenceState"`
}
type TestHistoryPage struct {
	Items   []TestHistoryItem `json:"items"`
	Cursor  string            `json:"cursor,omitempty"`
	Scanned int               `json:"scanned"`
	Partial bool              `json:"partial"`
}

type testHistoryCursor struct {
	Position string    `json:"position"`
	Expires  time.Time `json:"expiresAt"`
}

func (s *Service) TestHistory(ctx context.Context, user, project, suite, id, cursor string, limit int) (TestHistoryPage, error) {
	return s.ApplicationTestHistory(ctx, user, project, suite, id, "", cursor, limit)
}

func (s *Service) ApplicationTestHistory(ctx context.Context, user, project, suite, id, application, cursor string, limit int) (TestHistoryPage, error) {
	result := TestHistoryPage{Items: []TestHistoryItem{}}
	if !domain.ValidID(project) || !domain.ValidID(suite) || !domain.ValidID(id) || application != "" && !domain.ValidID(application) || limit < 1 || limit > 100 {
		return result, errors.New("test history requires valid identifiers and a page limit from 1 through 100")
	}
	for _, check := range [][2]string{{"", "runs.read"}, {"suite/" + suite, "catalog.read"}, {"test/" + suite + "/" + id, "catalog.read"}} {
		if err := s.Auth.Check(ctx, user, project, check[0], check[1]); err != nil {
			return result, err
		}
	}
	scope := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d", user, project, suite, id, limit) + "\x00" + application
	raw, err := paging.Open(ctx, s.Store, "test-history-settings", scope, cursor)
	if err != nil {
		return result, err
	}
	position := ""
	if raw != "" {
		var value testHistoryCursor
		if json.Unmarshal([]byte(raw), &value) != nil || !time.Now().Before(value.Expires) {
			return result, errors.New("test history cursor expired; refresh the newest history")
		}
		position = value.Position
	}
	for result.Scanned < 500 && len(result.Items) < limit {
		page, err := s.Store.ListNewest(ctx, "runs", project, position, min(50, limit-len(result.Items), 500-result.Scanned))
		if err != nil {
			return result, err
		}
		for _, row := range page.Items {
			result.Scanned++
			value, err := storage.Decode[Run](row)
			if err != nil {
				return result, err
			}
			if value.Project != project || value.Request.Suite != suite {
				continue
			}
			title := ""
			found := false
			for _, test := range value.Tests {
				if test.ID == id && TestInApplication(value, test, application) {
					title = test.Title
					found = true
					break
				}
			}
			if !found {
				continue
			}
			if err := s.Auth.Check(ctx, user, project, "run/"+value.ID, "runs.read"); errors.Is(err, authorization.ErrForbidden) {
				continue
			} else if err != nil {
				return result, err
			}
			value = NormalizeTimeouts(value)
			item := TestHistoryItem{Run: value.ID, Release: value.Release.ID, Title: title, CreatedAt: value.CreatedAt, FinishedAt: value.FinishedAt, Location: value.Request.Location, State: value.State, Outcome: value.Outcome, TestState: "not-recorded", EvidenceState: value.EvidenceState}
			if value.Environment != nil {
				item.Environment = value.Environment.Name
			}
			for _, attempt := range value.Results {
				if attempt.TestID == id {
					item.TestState, item.Attempt, item.Error = attempt.State, attempt.Attempt, attempt.Error
					duration := attempt.DurationMs
					item.Duration = &duration
					break
				}
			}
			result.Items = append(result.Items, item)
		}
		position = page.Cursor
		if position == "" {
			break
		}
	}
	if position != "" {
		data, _ := json.Marshal(testHistoryCursor{Position: position, Expires: time.Now().UTC().Add(15 * time.Minute)})
		result.Cursor, err = paging.Seal(ctx, s.Store, "test-history-settings", scope, string(data))
	}
	result.Partial = result.Scanned >= 500 && position != ""
	return result, err
}
