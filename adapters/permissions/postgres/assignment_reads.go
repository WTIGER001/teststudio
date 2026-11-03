package postgres

import (
	"context"
	"errors"
	permissions "github.com/wtiger001/go-permissions"
)

type assignmentReads struct{ permissions.PermissionStore }

var errAssignmentOnly = errors.New("only assignment changes are supported in this transaction")

func (assignmentReads) CreateRole(context.Context, permissions.Role) error { return errAssignmentOnly }
func (assignmentReads) UpdateRole(context.Context, permissions.Role) error { return errAssignmentOnly }
func (assignmentReads) DeleteRole(context.Context, string) error           { return errAssignmentOnly }
func (assignmentReads) AddRoleInheritance(context.Context, string, string) error {
	return errAssignmentOnly
}
func (assignmentReads) CreateGrant(context.Context, permissions.Grant) error {
	return errAssignmentOnly
}
func (assignmentReads) DeleteGrantsForOwner(context.Context, permissions.PrincipalKind, string) error {
	return errAssignmentOnly
}
func (assignmentReads) DisableBuiltInRole(context.Context, string) error { return errAssignmentOnly }
func (assignmentReads) EnableBuiltInRole(context.Context, string) error  { return errAssignmentOnly }
