package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/storage"
)

type runEvidenceFilter struct {
	Test    string `json:"test"`
	Attempt int    `json:"attempt"`
}

func evidenceFilter(query url.Values) (runEvidenceFilter, error) {
	value := runEvidenceFilter{Test: query.Get("test")}
	if value.Test != "" && !domain.ValidID(value.Test) {
		return value, errors.New("evidence filtering requires a valid test identifier")
	}
	if query.Get("attempt") != "" {
		attempt, err := strconv.Atoi(query.Get("attempt"))
		if err != nil || attempt < 1 || attempt > 3 || value.Test == "" {
			return value, errors.New("attempt filtering requires a test identifier and an attempt from 1 through 3")
		}
		value.Attempt = attempt
	}
	return value, nil
}

func (s *Server) filteredRunEvidence(ctx context.Context, actor, project, run, kind, cursor string, limit int, filter runEvidenceFilter) (storage.Page, error) {
	if limit < 1 || limit > 1000 {
		return storage.Page{}, errors.New("evidence page limit must be between 1 and 1000")
	}
	if filter.Test == "" {
		return s.Store.List(ctx, kind+"/"+run, project, cursor, limit)
	}
	scopeData, _ := json.Marshal([]any{actor, project, run, kind, filter})
	digest := sha256.Sum256(scopeData)
	scope := hex.EncodeToString(digest[:])
	type position struct {
		Scope  string `json:"scope"`
		Native string `json:"native"`
	}
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return storage.Page{}, errors.New("invalid evidence cursor")
		}
		var value position
		if json.Unmarshal(data, &value) != nil || value.Scope != scope || value.Native == "" {
			return storage.Page{}, errors.New("evidence cursor does not match the selected test and attempt")
		}
		cursor = value.Native
	}
	result := storage.Page{Items: []storage.Record{}}
	for scanned := 0; scanned < 200 && len(result.Items) < limit; {
		page, err := s.Store.List(ctx, kind+"/"+run, project, cursor, min(limit-len(result.Items), 200-scanned))
		if err != nil {
			return result, err
		}
		for _, row := range page.Items {
			var value struct {
				Test    string `json:"testId"`
				Attempt int    `json:"attempt"`
			}
			if err := json.Unmarshal(row.Data, &value); err != nil {
				return result, err
			}
			if value.Test == filter.Test && (filter.Attempt == 0 || value.Attempt == filter.Attempt) {
				result.Items = append(result.Items, row)
			}
		}
		scanned += len(page.Items)
		cursor = page.Cursor
		if cursor == "" {
			return result, nil
		}
	}
	data, _ := json.Marshal(position{Scope: scope, Native: cursor})
	result.Cursor = base64.RawURLEncoding.EncodeToString(data)
	return result, nil
}
