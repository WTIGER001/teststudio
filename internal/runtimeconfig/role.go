package runtimeconfig

import "errors"

type ServiceRole struct {
	API       bool
	Artifacts bool
	Worker    bool
}

func ParseServiceRole(value string) (ServiceRole, error) {
	switch value {
	case "all":
		return ServiceRole{API: true, Artifacts: true}, nil
	case "api":
		return ServiceRole{API: true}, nil
	case "artifacts":
		return ServiceRole{Artifacts: true}, nil
	case "worker":
		return ServiceRole{Worker: true}, nil
	default:
		return ServiceRole{}, errors.New("serve-role must be all, api, artifacts, or worker")
	}
}
