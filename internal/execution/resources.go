package execution

import "context"

type Reference struct {
	ID           string `json:"id"`
	Installation string `json:"installationId"`
	Project      string `json:"projectId"`
	Run          string `json:"runId"`
	Test         string `json:"testId"`
	Attempt      int    `json:"attempt"`
	Worker       string `json:"workerId"`
	Profile      string `json:"profile"`
	Container    string `json:"container"`
	Gateway      string `json:"gateway,omitempty"`
	Volume       string `json:"volume"`
}

type ManagedExecutor interface {
	CodedExecutor
	Reference(Reference) Reference
	Cleanup(context.Context, Reference) error
	Recover(context.Context, Reference, CodedInput) CodedReport
	Inventory(context.Context, string) ([]Reference, error)
}
