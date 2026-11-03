package recording

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const Lifetime = 20 * time.Minute
const ContentLimit = 256 << 10
const PlaywrightVersion = "1.64.1"

var ErrBusy = errors.New("a browser recording is already active on this computer")

type Input struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	TestID string `json:"testId"`
	Title  string `json:"title"`
}

type Session struct {
	Input
	Project    string    `json:"projectId"`
	Suite      string    `json:"suiteId"`
	Actor      string    `json:"actor"`
	State      string    `json:"state"`
	Code       string    `json:"code,omitempty"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	DeadlineAt time.Time `json:"deadlineAt"`
}

type Snapshot struct {
	State string
	Code  string
	Error string
}

type Driver interface {
	Start(context.Context, string, string) error
	Read(context.Context, string) (Snapshot, error)
	Stop(context.Context, string) (Snapshot, error)
}

func ValidURL(value string) bool {
	u, err := url.Parse(value)
	return len(value) > 0 && len(value) <= 2048 && err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && !strings.ContainsFunc(value, unicode.IsControl)
}

func Bucket() string { return "browser-recordings" }

func active(state string) bool { return state == "starting" || state == "recording" }
