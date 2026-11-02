package domain

import (
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
)

type AuditEvent struct {
	ID       string    `json:"id"`
	Actor    string    `json:"actor"`
	Project  string    `json:"projectId,omitempty"`
	Action   string    `json:"action"`
	Resource string    `json:"resource"`
	At       time.Time `json:"at"`
	Before   any       `json:"before,omitempty"`
	After    any       `json:"after,omitempty"`
}

func Audit(actor, project, action, resource string, before, after any) (storage.Change, error) {
	e := AuditEvent{ID: NewID(), Actor: actor, Project: project, Action: action, Resource: resource, At: time.Now().UTC(), Before: before, After: after}
	return storage.Put("audit", e.ID, project, 0, e)
}
