package authorization

import (
	"context"

	permissions "github.com/wtiger001/go-permissions"
)

type AssignmentPage struct {
	Items  []permissions.RoleAssignmentHit
	Cursor string
}

type AssignmentReader interface {
	AssignmentPage(context.Context, string, int) (AssignmentPage, error)
}
