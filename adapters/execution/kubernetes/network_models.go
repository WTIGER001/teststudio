package kubernetes

import (
	"context"
	"encoding/json"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type GatewayConfig struct {
	Image               string `json:"image"`
	ServiceAccount      string `json:"serviceAccount"`
	TLSSecret           string `json:"tlsSecret"`
	ExchangeTrustSecret string `json:"exchangeTrustSecret,omitempty"`
	CAFile              string `json:"caFile,omitempty"`
}

type GatewayAuthority interface {
	PrepareGateway(context.Context, execution.Reference, workerexchange.GatewaySettings) (workerexchange.GatewayTicket, error)
	BindGateway(context.Context, execution.Reference, string) error
	State(context.Context, execution.Reference) (workerexchange.State, error)
}

type networkResource struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	SpecHash  string `json:"specHash"`
	UID       string `json:"uid,omitempty"`
	Attempted bool   `json:"attempted,omitempty"`
}

type networkDispatch struct {
	WorkerAttempted bool              `json:"workerAttempted,omitempty"`
	Image           string            `json:"image"`
	Resources       []networkResource `json:"resources"`
	Failed          bool              `json:"failed,omitempty"`
}

type networkObject struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   metadata        `json:"metadata"`
	Spec       map[string]any  `json:"spec"`
	Status     json.RawMessage `json:"status,omitempty"`
}

func (o networkObject) String() string   { return "owned Kubernetes " + o.Kind + " " + o.Metadata.Name }
func (o networkObject) GoString() string { return o.String() }
