package workerprotocol

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
)

type DiscoveryInput struct {
	Project      string `json:"projectId"`
	Suite        string `json:"suiteId"`
	Runner       string `json:"runner"`
	Image        string `json:"image"`
	SourceDigest string `json:"sourceDigest"`
}

type DiscoveryReport struct {
	Raw          json.RawMessage `json:"raw,omitempty"`
	Failure      string          `json:"failure,omitempty"`
	Log          string          `json:"log,omitempty"`
	LogTruncated bool            `json:"logTruncated,omitempty"`
}

var discoveryImage = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64}$`)
var sourceDigest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func ValidateDiscoveryInput(input DiscoveryInput) error {
	if !domain.ValidID(input.Project) || !domain.ValidID(input.Suite) || input.Runner != "go" && input.Runner != "playwright" || len(input.Image) > 1024 || !discoveryImage.MatchString(input.Image) || !sourceDigest.MatchString(input.SourceDigest) || strings.Contains(input.Image, "..") {
		return ErrInvalid
	}
	return nil
}
