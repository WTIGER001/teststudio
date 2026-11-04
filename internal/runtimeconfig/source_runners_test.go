package runtimeconfig

import (
	"slices"
	"testing"
)

func TestSourceRunnersMatchTheConfiguredBuilderProfiles(t *testing.T) {
	for _, value := range []string{"go", "playwright", "go,playwright", " playwright , go "} {
		runners, err := SourceRunners(value)
		if err != nil || len(runners) < 1 || len(runners) > 2 {
			t.Fatal(value, runners, err)
		}
		if value == "go" && !slices.Equal(runners, []string{"go"}) {
			t.Fatal("single-profile configuration advertised another builder", runners)
		}
	}
	for _, value := range []string{"", "go,", "go,go", "node", "go,PLAYWRIGHT", "https", "playwright,go,playwright"} {
		if _, err := SourceRunners(value); err == nil {
			t.Fatal("unsupported or ambiguous builder profile accepted", value)
		}
	}
}
