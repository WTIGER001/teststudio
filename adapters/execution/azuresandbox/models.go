package azuresandbox

import (
	"encoding/json"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
)

type DiskImage struct {
	DefinitionHash string   `json:"definitionHash"`
	ID             string   `json:"id"`
	Base           string   `json:"base"`
	State          string   `json:"state"`
	Entrypoint     []string `json:"entrypoint"`
	Command        []string `json:"command"`
}

type Snapshot struct {
	ID             string `json:"id"`
	State          string `json:"state"`
	DefinitionHash string `json:"definitionHash"`
}

type Proof struct {
	Reference       execution.Reference `json:"reference"`
	Image           string              `json:"image"`
	SourceDigest    string              `json:"sourceDigest"`
	Sandbox         string              `json:"sandbox"`
	DiskImage       string              `json:"diskImage"`
	Runner          string              `json:"runner"`
	Operation       string              `json:"operation,omitempty"`
	Deadline        time.Time           `json:"deadline"`
	TicketHash      string              `json:"ticketHash"`
	EnvironmentHash string              `json:"environmentHash"`
	Fingerprint     string              `json:"fingerprint"`
}

type sandboxPage struct {
	Value    []json.RawMessage `json:"value"`
	NextLink string            `json:"nextLink"`
}

type imageResponse struct {
	Labels     map[string]string `json:"labels"`
	Dockerfile string            `json:"dockerfileContent"`
	ID         string            `json:"id"`
	Image      struct {
		Base       string   `json:"base"`
		Entrypoint []string `json:"entrypoint"`
		Command    []string `json:"cmd"`
	} `json:"image"`
	Status struct {
		State string `json:"state"`
	} `json:"status"`
}
