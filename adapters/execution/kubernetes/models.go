package kubernetes

import (
	"context"
	"net/http"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

const Bucket = "kubernetes-dispatches"
const workerLabel = "teststudio.io/worker"
const ownerAnnotation = "teststudio.io/reference"
const specAnnotation = "teststudio.io/spec"
const deadlineAnnotation = "teststudio.io/deadline"
const secondsAnnotation = "teststudio.io/active-seconds"

type Peer struct {
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type Config struct {
	Gateway          *GatewayConfig    `json:"gateway,omitempty"`
	APIOrigin        string            `json:"apiOrigin"`
	Namespace        string            `json:"namespace"`
	ServiceAccount   string            `json:"serviceAccount"`
	TokenFile        string            `json:"tokenFile"`
	CAFile           string            `json:"caFile"`
	Architecture     string            `json:"architecture"`
	RuntimeClass     string            `json:"runtimeClass"`
	PolicyName       string            `json:"policyName"`
	ExchangeOrigin   string            `json:"exchangeOrigin"`
	ExchangePeer     Peer              `json:"exchangePeer"`
	ExchangePort     int               `json:"exchangePort,omitempty"`
	DNSPeer          Peer              `json:"dnsPeer"`
	ImagePullSecrets []string          `json:"imagePullSecrets"`
	NodeSelector     map[string]string `json:"nodeSelector,omitempty"`
}

type Inspector interface {
	InspectImage(context.Context, string) (hostedexecution.Image, error)
}

type Options struct {
	GatewayAuthority GatewayAuthority
	DiscoveryCleanup func(context.Context, execution.Reference) error
	Config           Config
	Store            *storage.Store
	Inspector        Inspector
	Transport        http.RoundTripper
	Token            func(context.Context) (string, error)
}

type Provider struct {
	gatewayAuthority GatewayAuthority
	gatewayCA        []byte
	discoveryCleanup func(context.Context, execution.Reference) error
	config           Config
	store            *storage.Store
	inspector        Inspector
	token            func(context.Context) (string, error)
	http             *http.Client
	closeRegistry    func()
}

type dispatch struct {
	Network       *networkDispatch        `json:"network,omitempty"`
	Scope         string                  `json:"scope"`
	Reference     execution.Reference     `json:"reference"`
	Image         string                  `json:"image"`
	Deadline      time.Time               `json:"deadline"`
	Fingerprint   string                  `json:"fingerprint"`
	SpecHash      string                  `json:"specHash"`
	ActiveSeconds int                     `json:"activeSeconds"`
	UID           string                  `json:"uid,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
	GuardUntil    time.Time               `json:"guardUntil"`
	Rejected      bool                    `json:"rejected"`
	Stopping      bool                    `json:"stopping"`
	Removed       bool                    `json:"removed"`
	Terminal      *hostedexecution.Status `json:"terminal,omitempty"`
}

type metadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	UID               string            `json:"uid,omitempty"`
	Version           string            `json:"resourceVersion,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	DeletionTimestamp *time.Time        `json:"deletionTimestamp,omitempty"`
	OwnerReferences   []any             `json:"ownerReferences,omitempty"`
}

type pod struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   metadata       `json:"metadata"`
	Spec       map[string]any `json:"spec"`
	Status     struct {
		Phase      string `json:"phase"`
		Containers []struct {
			Name     string `json:"name"`
			Ready    bool   `json:"ready"`
			Restarts int    `json:"restartCount"`
			State    struct {
				Terminated *struct {
					Exit int `json:"exitCode"`
				} `json:"terminated,omitempty"`
			} `json:"state"`
		} `json:"containerStatuses"`
	} `json:"status,omitempty"`
}

var _ hostedexecution.Provider = (*Provider)(nil)
