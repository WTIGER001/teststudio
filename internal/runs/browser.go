package runs

import (
	"errors"
	"net/url"

	"github.com/wtiger001/teststudio/internal/apipolicy"
	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/projects"
)

func browserTarget(environment *projects.Environment, artifact catalog.Artifact, test catalog.Test) (*browserpolicy.Target, error) {
	if environment == nil || artifact.Browser == nil {
		return nil, errors.New("recorded browser tests require a registered target environment")
	}
	if err := browserpolicy.Validate(artifact.Browser); err != nil {
		return nil, err
	}
	if artifact.Browser.Requests != nil && test.Type != "api" {
		return nil, errors.New("a request-restricted profile requires API tests")
	}
	if test.Type == "api" && environment.Protected && !apipolicy.ReadOnly(artifact.Browser.Requests) {
		return nil, errors.New("protected API execution requires a reviewed read-only request policy")
	}
	if err := apipolicy.ValidateBindings(artifact.Browser.Requests, test.RequiredBindings); err != nil {
		return nil, err
	}
	origin, err := browserpolicy.Origin(environment.Endpoints[artifact.Browser.Endpoint])
	if err != nil {
		return nil, err
	}
	u, _ := url.Parse(origin)
	if u.Scheme == "http" && !environment.AllowHTTP && environment.Stage != "local" {
		return nil, errors.New("the selected browser environment does not permit HTTP")
	}
	target := &browserpolicy.Target{Origin: origin, AllowPrivate: environment.AllowPrivate || environment.Stage == "local", Requests: artifact.Browser.Requests}
	if len(test.RequiredBindings) > 0 {
		if !domain.ValidID(environment.Credentials[test.RequiredBindings[0]]) {
			return nil, errors.New("the selected environment needs the declared API credential reference")
		}
		if u.Scheme != "https" && environment.Stage != "local" {
			return nil, errors.New("API credential injection requires HTTPS outside local environments")
		}
		target.Authenticated, target.AllowCredentialHTTP = true, u.Scheme == "http" && environment.Stage == "local"
	}
	return target, browserpolicy.ValidateTarget(*target)
}
