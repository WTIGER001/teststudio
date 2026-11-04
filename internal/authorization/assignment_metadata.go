package authorization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	permissions "github.com/wtiger001/go-permissions"
)

type AssignmentIdentity struct {
	PrincipalKind permissions.PrincipalKind `json:"principalKind"`
	PrincipalID   string                    `json:"principalId"`
	RoleID        string                    `json:"roleId"`
	Bindings      map[string]any            `json:"bindingValues"`
}

type AssignmentState struct {
	AssignmentIdentity
	Present bool `json:"present"`
}

func (a AssignmentIdentity) ID() string {
	raw, _ := json.Marshal(a)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (a AssignmentIdentity) Principal() permissions.PrincipalRef {
	return permissions.PrincipalRef{Kind: a.PrincipalKind, ID: a.PrincipalID}
}
