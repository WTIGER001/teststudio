package execution

import (
	"encoding/json"
	"errors"
	"regexp"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/capture"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/workerwire"
)

const MaxWorkItemBytes = workerwire.MaxWorkItemBytes

var imageDigest = regexp.MustCompile(`^(sha256:[a-f0-9]{64}|[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64})$`)

func MarshalWorkItem(input CodedInput) ([]byte, error) {
	if !domain.ValidID(input.RunID) || !domain.ValidID(input.Test.ID) || (input.Test.Runner != "go" && input.Test.Runner != "playwright") || input.Test.Selector == "" || input.TimeoutMs < 1 || input.TimeoutMs > 3600000 || !imageDigest.MatchString(input.Artifact.Image) || input.Artifact.Discovered[input.Test.ID] != input.Test.Selector {
		return nil, errors.New("worker input requires a selected test, immutable artifact, discovered selector, and bounded timeout")
	}
	if (input.Browser != nil) != (input.Artifact.Browser != nil) {
		return nil, errors.New("worker input target does not match the source profile")
	}
	if input.Browser != nil {
		if err := browserpolicy.ValidateTarget(*input.Browser); err != nil {
			return nil, err
		}
	}
	input.Capture = capture.Normalize(input.Capture)
	if err := capture.Validate(input.Capture, capture.DefaultLimits()); err != nil {
		return nil, err
	}
	input.Test = catalog.Test{ID: input.Test.ID, Selector: input.Test.Selector, Runner: input.Test.Runner, Type: input.Test.Type, RequiredBindings: input.Test.RequiredBindings}
	input.Artifact = catalog.Artifact{Image: input.Artifact.Image, Browser: input.Artifact.Browser, SourceProject: input.Artifact.SourceProject, SourceSnapshot: input.Artifact.SourceSnapshot, SourceRevision: input.Artifact.SourceRevision, Discovered: map[string]string{input.Test.ID: input.Test.Selector}}
	data, err := json.Marshal(input)
	if err != nil || len(data) > MaxWorkItemBytes {
		return nil, errors.New("worker input is invalid or exceeds 4 MiB")
	}
	return data, nil
}
