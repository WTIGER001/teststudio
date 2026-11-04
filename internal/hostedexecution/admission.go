package hostedexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"time"

	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/gosuite"
)

var distribution = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[a-f0-9]{64}$`)
var name = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var runtime = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$`)
var selector = regexp.MustCompile(`^@[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func (e *Executor) Profile() string { return e.Runner + "-" + e.Name + "-isolated-v1" }

func (e *Executor) configured() bool {
	origin, err := url.Parse(e.ExchangeOrigin)
	return e.Provider != nil && e.Exchange != nil && e.ApproveArtifact != nil && name.MatchString(e.Name) && (e.Runner == "go" || e.Runner == "playwright") && e.PollInterval >= 0 && e.PollInterval <= 10*time.Second && err == nil && origin.Scheme == "https" && origin.Hostname() != "" && origin.User == nil && origin.Path == "" && origin.RawQuery == "" && !origin.ForceQuery && origin.Fragment == ""
}

func (e *Executor) image(ctx context.Context, artifact catalog.Artifact) (Image, error) {
	if !e.configured() || !distribution.MatchString(artifact.Image) || artifact.Browser != nil {
		return Image{}, ErrConfiguration
	}
	if err := e.ApproveArtifact(ctx, artifact); err != nil {
		return Image{}, ErrConfiguration
	}
	check, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	image, err := e.Provider.InspectImage(check, artifact.Image)
	if err != nil {
		return Image{}, ErrUnavailable
	}
	if image.Reference != artifact.Image || image.Runner != e.Runner || !runtime.MatchString(image.Runtime) || image.OS != "linux" || (image.Architecture != "amd64" && image.Architecture != "arm64") || image.Supervisor != "worker-exchange/v1" {
		return Image{}, ErrConfiguration
	}
	if artifact.SourceProject != "" && image.Project != artifact.SourceProject || artifact.Provenance != nil && artifact.Provenance.SnapshotDigest != "" && image.SourceDigest != artifact.Provenance.SnapshotDigest {
		return Image{}, ErrConfiguration
	}
	return image, nil
}

func (e *Executor) Admit(ctx context.Context, test catalog.Test, artifact catalog.Artifact) error {
	if test.Runner != e.Runner || !domain.ValidID(test.ID) || artifact.Discovered[test.ID] != test.Selector || len(test.RequiredBindings) != 0 || artifact.Browser != nil {
		return ErrConfiguration
	}
	if e.Runner == "go" {
		if _, err := gosuite.Parse(test.Selector); err != nil {
			return ErrConfiguration
		}
	} else if !selector.MatchString(test.Selector) {
		return ErrConfiguration
	}
	_, err := e.image(ctx, artifact)
	return err
}

func (e *Executor) TimingProfile(ctx context.Context, artifact catalog.Artifact) (string, error) {
	image, err := e.image(ctx, artifact)
	if err != nil {
		return "", err
	}
	return e.Profile() + ":" + image.Runtime + ":" + image.OS + "-" + image.Architecture, nil
}

func (e *Executor) Reference(ref execution.Reference) execution.Reference {
	ref.Profile = e.Profile()
	id := sha256.Sum256([]byte(ref.ID))
	ref.Container = "teststudio-job-" + hex.EncodeToString(id[:20])
	ref.Volume = ""
	ref.Gateway = ""
	return ref
}

func (e *Executor) validReference(ref execution.Reference) bool {
	return domain.ValidID(ref.ID) && domain.ValidID(ref.Installation) && domain.ValidID(ref.Project) && domain.ValidID(ref.Run) && domain.ValidID(ref.Test) && domain.ValidID(ref.Worker) && ref.Attempt >= 1 && ref.Attempt <= 3 && ref == e.Reference(ref)
}
