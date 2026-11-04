package runs

import (
	"errors"
	"reflect"
	"slices"

	"github.com/wtiger001/teststudio/internal/projects"
)

func currentEnvironmentPolicy(run Run, current projects.Environment, suite projects.Suite) error {
	previous := run.Environment
	if previous == nil || current.Application != previous.Application || current.Stage != previous.Stage || current.AllowHTTP != previous.AllowHTTP || current.AllowPrivate != previous.AllowPrivate || !reflect.DeepEqual(current.Credentials, previous.Credentials) || len(current.Locations) > 0 && !slices.Contains(current.Locations, run.Request.Location) || len(suite.Applications) > 0 && !slices.Contains(suite.Applications, current.Application) {
		return errors.New("the target policy changed after review; create a new run with the current environment")
	}
	if run.Release.Draft.Artifact.Browser != nil && !reflect.DeepEqual(current.Endpoints, previous.Endpoints) {
		return errors.New("the browser target policy changed after review; create a new run with the current environment")
	}
	return nil
}
