package trends

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/authorization"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Service struct {
	Store *storage.Store
	Auth  *authorization.Service
}
type cursorValue struct {
	Scope    string `json:"scope"`
	Position string `json:"position"`
}

func (s *Service) Read(ctx context.Context, user, project string, filter Filter, cursor string, limit int) (Report, error) {
	if !domain.ValidID(project) {
		return Report{}, errors.New("valid project identifier is required")
	}
	if _, err := s.Store.Get(ctx, "projects", project); err != nil {
		return Report{}, err
	}
	if err := s.Auth.Check(ctx, user, project, "", "runs.read"); err != nil {
		return Report{}, err
	}
	if limit < 1 || limit > 500 {
		return Report{}, errors.New("trend scan limit must be between 1 and 500")
	}
	if filter.Since.IsZero() || filter.Until.IsZero() || filter.Since.After(filter.Until) || filter.Until.Sub(filter.Since) > 366*24*time.Hour {
		return Report{}, errors.New("trend period must be ordered and no longer than 366 days")
	}
	if len(filter.Category) > 256 || len(filter.Location) > 128 {
		return Report{}, errors.New("trend filter exceeds its length limit")
	}
	if filter.Suite != "" && !domain.ValidID(filter.Suite) || filter.Test != "" && !domain.ValidID(filter.Test) || filter.Environment != "" && !domain.ValidID(filter.Environment) {
		return Report{}, errors.New("invalid trend resource identifier")
	}
	if filter.Test != "" && filter.Suite == "" {
		return Report{}, errors.New("a test trend requires its suite identifier")
	}
	encoded, _ := json.Marshal(struct {
		Project, User string
		Filter        Filter
		Limit         int
	}{project, user, filter, limit})
	hash := sha256.Sum256(encoded)
	scope := hex.EncodeToString(hash[:])
	position := ""
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		var value cursorValue
		if err != nil || json.Unmarshal(data, &value) != nil || value.Scope != scope {
			return Report{}, errors.New("trend cursor does not match the request")
		}
		position = value.Position
	}
	page, err := s.Store.ListNewest(ctx, "runs", project, position, limit)
	if err != nil {
		return Report{}, err
	}
	input := []runs.Run{}
	for _, row := range page.Items {
		run, err := storage.Decode[runs.Run](row)
		if err != nil {
			return Report{}, err
		}
		if run.Project != project {
			return Report{}, errors.New("run history contains an inconsistent project")
		}
		if !inPeriod(run, filter) {
			continue
		}
		if err := s.Auth.Check(ctx, user, project, "run/"+run.ID, "runs.read"); err != nil {
			if errors.Is(err, authorization.ErrForbidden) {
				continue
			}
			return Report{}, err
		}
		input = append(input, run)
	}
	report := Aggregate(input, filter)
	report.Project = project
	report.Partial = page.Cursor != ""
	if report.Partial {
		data, _ := json.Marshal(cursorValue{Scope: scope, Position: page.Cursor})
		report.Cursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return report, nil
}
