package recording

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/projects"
	"github.com/wtiger001/teststudio/internal/storage"
)

type Step struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type CheckTarget struct {
	Label         string         `json:"label"`
	Matches       int            `json:"matches"`
	Text          string         `json:"text"`
	ElementLabel  string         `json:"elementLabel,omitempty"`
	Value         string         `json:"value,omitempty"`
	Capabilities  []string       `json:"capabilities,omitempty"`
	Bounds        []TargetBounds `json:"bounds,omitempty"`
	ElementBounds []TargetBounds `json:"elementBounds,omitempty"`
}

type TargetBounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type View struct {
	Target   *CheckTarget `json:"target,omitempty"`
	Message  string       `json:"message,omitempty"`
	Image    string       `json:"image"`
	Width    int          `json:"width"`
	Height   int          `json:"height"`
	URL      string       `json:"url"`
	Mode     string       `json:"mode"`
	Sequence int          `json:"sequence"`
	Steps    []Step       `json:"steps"`
}

type Action struct {
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Count    int     `json:"count,omitempty"`
	Sequence int     `json:"sequence"`
	Action   string  `json:"action"`
	X        float64 `json:"x,omitempty"`
	Y        float64 `json:"y,omitempty"`
	Text     string  `json:"text,omitempty"`
	Key      string  `json:"key,omitempty"`
	Scope    string  `json:"scope,omitempty"`
}

type InteractiveDriver interface {
	Driver
	View(context.Context, string) (View, error)
	Interact(context.Context, string, Action) (View, error)
}

func (a Action) Valid() bool {
	if a.Sequence < 1 || a.Sequence > 10000 || math.IsNaN(a.X) || math.IsNaN(a.Y) || math.IsInf(a.X, 0) || math.IsInf(a.Y, 0) {
		return false
	}
	if a.Action != "resize" && (a.Width != 0 || a.Height != 0) || a.Action != "assert-add" && a.Count != 0 {
		return false
	}
	if a.Scope != "" && (a.Action != "assert-add" || !slices.Contains([]string{"at-least", "at-most", "exactly"}, a.Key) || !slices.Contains([]string{"element", "group"}, a.Scope)) {
		return false
	}
	switch a.Action {
	case "click":
		return a.X >= 0 && a.X < 1920 && a.Y >= 0 && a.Y < 1200 && a.Text == "" && a.Key == ""
	case "text":
		return a.X == 0 && a.Y == 0 && len(a.Text) > 0 && len(a.Text) <= 2048 && utf8.ValidString(a.Text) && a.Key == ""
	case "key":
		return a.X == 0 && a.Y == 0 && a.Text == "" && slices.Contains([]string{"Enter", "Tab", "Shift+Tab", "Backspace", "Delete", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Home", "End", "Escape", "ControlOrMeta+A"}, a.Key)
	case "scroll":
		return a.X == 0 && math.Abs(a.Y) <= 1000 && a.Text == "" && a.Key == ""
	case "resize":
		return a.Width >= 320 && a.Width <= 1920 && a.Height >= 240 && a.Height <= 1200 && a.X == 0 && a.Y == 0 && a.Text == "" && a.Key == ""
	case "assert-add":
		if a.X != 0 || a.Y != 0 {
			return false
		}
		if slices.Contains([]string{"text", "exact-text", "value"}, a.Key) {
			return a.Count == 0 && (a.Key == "value" || len(a.Text) > 0) && len(a.Text) <= 2048 && utf8.ValidString(a.Text)
		}
		if slices.Contains([]string{"visible", "enabled", "disabled", "checked", "unchecked"}, a.Key) {
			return a.Count == 0 && a.Text == ""
		}
		return a.Text == "" && (a.Key == "at-least" && a.Count >= 1 || slices.Contains([]string{"exactly", "at-most"}, a.Key) && a.Count >= 0) && a.Count <= 10000
	case "assert-visible", "assert-pick", "assert-url", "assert-title", "record":
		return a.X == 0 && a.Y == 0 && a.Text == "" && a.Key == ""
	}
	return false
}

func (s *Service) Interactive(ctx context.Context, user, project, suite, id string, input *Action) (View, error) {
	s.operation.Lock()
	defer s.operation.Unlock()
	value, row, err := s.owned(ctx, user, project, suite, id, true)
	if err != nil {
		return View{}, err
	}
	driver, ok := s.Driver.(InteractiveDriver)
	if !ok || value.Mode != "embedded" || value.State != "recording" || !time.Now().Before(value.DeadlineAt) {
		return View{}, errors.New("this recording browser is closed; review its saved steps or start a new recording")
	}
	suiteRow, err := s.Store.Get(ctx, projects.Bucket("suites", project), suite)
	if err != nil {
		return View{}, err
	}
	definition, err := storage.Decode[projects.Suite](suiteRow)
	if err != nil {
		return View{}, err
	}
	if definition.Archived {
		return View{}, errors.New("this suite is archived; restore it before continuing the recording")
	}
	if input == nil {
		return driver.View(ctx, id)
	}
	if !input.Valid() {
		return View{}, errors.New("choose a supported action inside the recording browser")
	}
	view, err := driver.Interact(ctx, id, *input)
	if err != nil {
		return View{}, err
	}
	snapshot, err := s.Driver.Read(ctx, id)
	if err == nil {
		_, err = s.finish(ctx, row, value, snapshot)
	}
	return view, err
}
