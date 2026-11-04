package runs

import (
	"errors"
	"net/url"
	"reflect"
	"slices"

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

func currentBrowserEnvironment(run Run, current projects.Environment, suite projects.Suite) error {
	if run.Release.Draft.Artifact.Browser == nil {
		return nil
	}
	previous := run.Environment
	if previous == nil || current.Application != previous.Application || current.Stage != previous.Stage || current.AllowHTTP != previous.AllowHTTP || current.AllowPrivate != previous.AllowPrivate || !reflect.DeepEqual(current.Endpoints, previous.Endpoints) || !reflect.DeepEqual(current.Credentials, previous.Credentials) || len(current.Locations) > 0 && !slices.Contains(current.Locations, run.Request.Location) || len(suite.Applications) > 0 && !slices.Contains(suite.Applications, current.Application) {
		return errors.New("the browser target policy changed after review; create a new run with the current environment")
	}
	return nil
}
