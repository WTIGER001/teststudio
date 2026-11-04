package runs

import (
	"slices"

	"github.com/wtiger001/teststudio/internal/catalog"
)

func TestInApplication(run Run, test catalog.Test, application string) bool {
	if application == "" {
		return true
	}
	if run.Request.Application != "" && run.Request.Application != application {
		return false
	}
	if run.Environment != nil && run.Environment.Application != application {
		return false
	}
	if test.Application != "" {
		return test.Application == application
	}
	return run.Request.Application == application || run.Environment != nil && run.Environment.Application == application || slices.Contains(run.Applications, application)
}

func InApplication(run Run, application string) bool {
	if application == "" {
		return true
	}
	if run.Request.Application != "" {
		return run.Request.Application == application
	}
	if run.Environment != nil {
		return run.Environment.Application == application
	}
	for _, test := range run.Tests {
		if TestInApplication(run, test, application) {
			return true
		}
	}
	return len(run.Tests) == 0 && slices.Contains(run.Applications, application)
}
