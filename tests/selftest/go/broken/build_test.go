package broken

import "testing"

func TestBuild(t *testing.T) { intentionallyUndefinedDiagnostic() }
