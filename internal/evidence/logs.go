package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/wtiger001/teststudio/internal/artifacts"
	"io"
	"slices"
	"strings"
)

type LogSegment struct {
	Artifact Artifact `json:"artifact"`
	Text     string   `json:"text"`
}
type LogPage struct {
	Segments  []LogSegment `json:"segments"`
	Cursor    string       `json:"cursor,omitempty"`
	Truncated bool         `json:"truncated"`
}

func (s *Service) Logs(ctx context.Context, project, run, stream, test, cursor string, tail bool, limit int) (LogPage, error) {
	if limit < 1 || limit > 100 {
		return LogPage{}, errors.New("log page limit must be between 1 and 100")
	}
	rows, err := s.Store.All(ctx, Bucket(run), project)
	if err != nil {
		return LogPage{}, err
	}
	entries := []Artifact{}
	for _, r := range rows {
		a, err := storageArtifact(r.Data)
		if err != nil {
			return LogPage{}, err
		}
		if a.Kind == "log" && (stream == "" || stream == a.Stream) && (test == "" || test == a.Test) {
			entries = append(entries, a)
		}
	}
	slices.SortFunc(entries, func(a, b Artifact) int {
		if n := a.CreatedAt.Compare(b.CreatedAt); n != 0 {
			return n
		}
		if n := strings.Compare(a.Worker, b.Worker); n != 0 {
			return n
		}
		if a.Sequence < b.Sequence {
			return -1
		}
		if a.Sequence > b.Sequence {
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
	data, _ := json.Marshal(struct {
		Project, Run, Stream, Test string
		Entries                    []Artifact
	}{project, run, stream, test, entries})
	hash := sha256.Sum256(data)
	version := hex.EncodeToString(hash[:])
	position := 0
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return LogPage{}, errors.New("invalid log cursor")
		}
		var value struct {
			Version  string
			Position int
		}
		if err := json.Unmarshal(data, &value); err != nil || value.Version != version || value.Position < 0 || value.Position > len(entries) {
			return LogPage{}, errors.New("log collection changed; reload before continuing")
		}
		position = value.Position
	}
	page := LogPage{Segments: []LogSegment{}}
	end := min(position+limit, len(entries))
	if tail {
		position = max(0, len(entries)-limit)
		end = len(entries)
		page.Truncated = position > 0
	}
	bytes := int64(0)
	selected := slices.Clone(entries[position:end])
	if tail {
		slices.Reverse(selected)
	}
	for _, a := range selected {
		segment := LogSegment{Artifact: a}
		if a.State == "available" {
			current, reader, err := s.Open(ctx, project, run, a.ID)
			segment.Artifact = current
			if err != nil && !errors.Is(err, artifacts.ErrNotFound) {
				return LogPage{}, err
			}
			if err == nil {
				body, readErr := io.ReadAll(io.LimitReader(reader, 64<<10))
				reader.Close()
				if readErr != nil {
					return LogPage{}, readErr
				}
				bytes += int64(len(body))
				if bytes > 1<<20 {
					page.Truncated = true
					break
				}
				segment.Text = string(body)
			}
		}
		page.Segments = append(page.Segments, segment)
	}
	if tail {
		slices.Reverse(page.Segments)
	}
	if !tail && position+len(page.Segments) < len(entries) {
		data, _ := json.Marshal(struct {
			Version  string
			Position int
		}{version, position + len(page.Segments)})
		page.Cursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, nil
}

func storageArtifact(data []byte) (Artifact, error) {
	var value Artifact
	err := json.Unmarshal(data, &value)
	return value, err
}
