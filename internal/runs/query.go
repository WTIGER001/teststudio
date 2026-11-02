package runs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

type Filter struct {
	Environment string    `json:"environment,omitempty"`
	Suite       string    `json:"suite,omitempty"`
	Application string    `json:"application,omitempty"`
	State       string    `json:"state,omitempty"`
	Stage       string    `json:"stage,omitempty"`
	Selection   string    `json:"selection,omitempty"`
	Since       time.Time `json:"since,omitempty"`
	Until       time.Time `json:"until,omitempty"`
}

type Summary struct {
	ID              string     `json:"id"`
	Project         string     `json:"projectId"`
	Suite           string     `json:"suiteId"`
	Environment     string     `json:"environmentId,omitempty"`
	EnvironmentName string     `json:"environmentName,omitempty"`
	Stage           string     `json:"stage,omitempty"`
	Protected       bool       `json:"protected"`
	State           string     `json:"state"`
	Outcome         string     `json:"outcome"`
	Phase           string     `json:"phase"`
	Counts          Counts     `json:"counts"`
	CreatedAt       time.Time  `json:"createdAt"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	EvidenceState   string     `json:"evidenceState"`
	Actor           string     `json:"actor"`
	Trigger         string     `json:"trigger"`
	Selection       string     `json:"selection"`
}

type Page struct {
	Items   []storage.Record `json:"items"`
	Cursor  string           `json:"cursor,omitempty"`
	Scanned int              `json:"scanned"`
	Partial bool             `json:"partial"`
}

func Summarize(v Run) Summary {
	s := Summary{ID: v.ID, Project: v.Project, Suite: v.Request.Suite, Environment: v.Request.Environment, State: v.State, Outcome: v.Outcome, Phase: v.Phase, Counts: v.Counts, CreatedAt: v.CreatedAt, FinishedAt: v.FinishedAt, EvidenceState: v.EvidenceState, Actor: v.Actor, Trigger: v.Trigger, Selection: "full"}
	if len(v.Request.IDs) > 0 {
		s.Selection = "selection"
	}
	if v.Environment != nil {
		s.EnvironmentName = v.Environment.Name
		s.Stage = v.Environment.Stage
		s.Protected = v.Environment.Protected
	}
	return s
}

func matches(v Run, f Filter) bool {
	if f.Environment == "none" {
		if v.Environment != nil {
			return false
		}
	} else if f.Environment != "" && v.Request.Environment != f.Environment {
		return false
	}
	if f.Suite != "" && v.Request.Suite != f.Suite {
		return false
	}
	if f.Application != "" && (v.Environment == nil || v.Environment.Application != f.Application) {
		return false
	}
	if f.State != "" && v.State != f.State {
		return false
	}
	if f.Stage != "" && (v.Environment == nil || v.Environment.Stage != f.Stage) {
		return false
	}
	if f.Selection != "" && Summarize(v).Selection != f.Selection {
		return false
	}
	return (f.Since.IsZero() || !v.CreatedAt.Before(f.Since)) && (f.Until.IsZero() || !v.CreatedAt.After(f.Until))
}

func (s *Service) List(ctx context.Context, user, project string, filter Filter, cursor string, limit int) (Page, error) {
	if err := s.Auth.Check(ctx, user, project, "", "runs.read"); err != nil {
		return Page{}, err
	}
	if limit < 1 || limit > 100 {
		return Page{}, errors.New("run page limit must be between 1 and 100")
	}
	queryHash := fingerprint(struct {
		Project string
		Filter  Filter
	}{project, filter})
	storeCursor := ""
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return Page{}, errors.New("invalid run cursor")
		}
		var v struct {
			Query    string
			Position string
		}
		if err := json.Unmarshal(data, &v); err != nil || v.Query != queryHash {
			return Page{}, errors.New("run cursor does not match the query")
		}
		storeCursor = v.Position
	}
	result := Page{Items: []storage.Record{}}
	for result.Scanned < 500 && len(result.Items) < limit {
		page, err := s.Store.ListNewest(ctx, "runs", project, storeCursor, min(50, limit-len(result.Items), 500-result.Scanned))
		if err != nil {
			return Page{}, err
		}
		for _, row := range page.Items {
			result.Scanned++
			v, err := storage.Decode[Run](row)
			if err != nil {
				return Page{}, err
			}
			if matches(v, filter) {
				row.Data, err = json.Marshal(Summarize(v))
				if err != nil {
					return Page{}, err
				}
				result.Items = append(result.Items, row)
			}
		}
		storeCursor = page.Cursor
		if storeCursor == "" {
			break
		}
	}
	if storeCursor != "" {
		data, _ := json.Marshal(struct {
			Query    string
			Position string
		}{queryHash, storeCursor})
		result.Cursor = base64.RawURLEncoding.EncodeToString(data)
	}
	result.Partial = result.Scanned >= 500 && storeCursor != ""
	return result, nil
}
