package runs

import (
	"errors"
	"net/url"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/projects"
)

func browserTarget(environment *projects.Environment, artifact catalog.Artifact, test catalog.Test) (*browserpolicy.Target, error) {
	if environment == nil || artifact.Browser == nil {
		return nil, errors.New("recorded browser tests require a registered target environment")
	}
	if err := browserpolicy.Validate(artifact.Browser); err != nil {
		return nil, err
	}
	if len(test.RequiredBindings) > 0 {
		return nil, errors.New("the single-origin browser profile does not inject credential bindings")
	}
	origin, err := browserpolicy.Origin(environment.Endpoints[artifact.Browser.Endpoint])
	if err != nil {
		return nil, err
	}
	u, _ := url.Parse(origin)
	if u.Scheme == "http" && !environment.AllowHTTP && environment.Stage != "local" {
		return nil, errors.New("the selected browser environment does not permit HTTP")
	}
	target := &browserpolicy.Target{Origin: origin, AllowPrivate: environment.AllowPrivate || environment.Stage == "local"}
	return target, browserpolicy.ValidateTarget(*target)
}
