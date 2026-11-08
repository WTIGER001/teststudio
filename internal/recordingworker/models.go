package recordingworker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/recording"
)

const LeaseBucket = "recording-worker-leases"
const PoolBucket = "recording-worker-pools"
const OwnerLifetime = 90 * time.Second

type Binding struct {
	ID       string    `json:"id"`
	Actor    string    `json:"actor"`
	Project  string    `json:"projectId"`
	Suite    string    `json:"suiteId"`
	URL      string    `json:"url"`
	Deadline time.Time `json:"deadlineAt"`
}

func Bind(s recording.Session) Binding {
	return Binding{ID: s.ID, Actor: s.Actor, Project: s.Project, Suite: s.Suite, URL: s.URL, Deadline: s.DeadlineAt}
}

func (b Binding) Fingerprint() string {
	data, _ := json.Marshal(b)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type Reference struct {
	Provider     string `json:"provider"`
	Container    string `json:"container"`
	Gateway      string `json:"gateway"`
	Volume       string `json:"volume"`
	Fingerprint  string `json:"fingerprint"`
	RuntimeImage string `json:"runtimeImage"`
	GatewayImage string `json:"gatewayImage"`
	PolicyHash   string `json:"policyHash"`
	NetworkHash  string `json:"networkHash,omitempty"`
}

type Runtime interface {
	Plan(Binding, browserpolicy.Target) (Reference, error)
	Start(context.Context, Binding, Reference, browserpolicy.Target) (recording.InteractiveDriver, error)
	Cleanup(context.Context, Binding, Reference) error
}

type Lease struct {
	Binding        Binding              `json:"binding"`
	Worker         string               `json:"worker"`
	Instance       string               `json:"instance"`
	Reference      Reference            `json:"reference"`
	Target         browserpolicy.Target `json:"target,omitempty"`
	Snapshot       recording.Snapshot   `json:"snapshot"`
	LastSeen       time.Time            `json:"lastSeenAt"`
	Sequence       int                  `json:"sequence"`
	InputHash      string               `json:"inputHash,omitempty"`
	Pending        bool                 `json:"pending,omitempty"`
	CleanupPending bool                 `json:"cleanupPending"`
	Closing        bool                 `json:"closing,omitempty"`
}

type Pool struct {
	Instance string    `json:"instance"`
	Until    time.Time `json:"until"`
}

type Request struct {
	Binding Binding           `json:"binding"`
	Action  *recording.Action `json:"input,omitempty"`
}

type Response struct {
	Snapshot recording.Snapshot `json:"snapshot"`
	View     *recording.View    `json:"view,omitempty"`
}
