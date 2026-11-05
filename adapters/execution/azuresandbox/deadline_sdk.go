package azuresandbox

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type DeadlineOptions struct {
	Config     DeadlineConfig
	Exchange   *workerexchange.Service
	Instances  Instances
	Credential azcore.TokenCredential
	Transport  policy.Transporter
	Now        func() time.Time
}

type SDKDeadlines struct {
	config    DeadlineConfig
	control   *Controller
	workflows *l.WorkflowsClient
	http      *deadlineHTTP
}

func NewSDKDeadlines(o DeadlineOptions) (*SDKDeadlines, error) {
	if o.Config.validate() != nil || o.Exchange == nil || o.Exchange.Store == nil || o.Exchange.Authorize == nil || o.Instances == nil || o.Credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	h := newDeadlineHTTP(o.Config, o.Transport)
	options := &arm.ClientOptions{DisableRPRegistration: true, ClientOptions: policy.ClientOptions{
		Transport: h, Retry: policy.RetryOptions{MaxRetries: -1},
		Cloud: cloud.Configuration{ActiveDirectoryAuthorityHost: "https://login.microsoftonline.com/", Services: map[cloud.ServiceName]cloud.ServiceConfiguration{cloud.ResourceManager: {Endpoint: "https://management.azure.com/", Audience: "https://management.core.windows.net/"}}},
	}}
	client, err := l.NewWorkflowsClient(o.Config.Worker.Subscription, o.Credential, options)
	if err != nil {
		h.Close()
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKDeadlines{config: o.Config, http: h, workflows: client, control: &Controller{config: o.Config.Worker, store: o.Exchange.Store, exchange: o.Exchange, instances: o.Instances, now: now}}, nil
}

func (c *SDKDeadlines) Close() { c.http.Close() }

func timerError(err error, mutation bool) error {
	if err == nil {
		return nil
	}
	var response *azcore.ResponseError
	if errors.As(err, &response) {
		if !mutation && response.StatusCode == http.StatusNotFound {
			return hostedexecution.ErrNotFound
		}
		if mutation {
			switch response.StatusCode {
			case 400, 401, 403, 404, 405, 413, 415, 422:
				return hostedexecution.ErrRejected
			}
		}
	}
	return controlError(err)
}

func (c *SDKDeadlines) load(ctx context.Context, s DeadlineSpec) (dispatch, error) {
	if c.config.validateSpec(s) != nil {
		return dispatch{}, hostedexecution.ErrOwnership
	}
	d, _, _, err := c.control.load(ctx, s.Proof.Reference)
	if err != nil {
		return d, err
	}
	if !d.DeadlineAttempted || d.Proof.Fingerprint != s.Proof.Fingerprint || d.TimerProfile != "" && d.TimerProfile != c.config.profile() {
		return d, hostedexecution.ErrOwnership
	}
	return d, nil
}

func (c *SDKDeadlines) get(ctx context.Context, s DeadlineSpec) (l.Workflow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.workflows.Get(ctx, c.config.ResourceGroup, c.config.name(s), nil)
	return result.Workflow, timerError(err, false)
}

func (c *SDKDeadlines) Ensure(ctx context.Context, s DeadlineSpec) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, err := c.load(ctx, s)
	if err != nil {
		return err
	}
	if d.DeadlineRemoving || d.DeadlineRemoved || d.TimerCreateRejected || d.Stopping {
		return hostedexecution.ErrRejected
	}
	if d.TimerCreateAttempted {
		return c.Verify(ctx, s)
	}
	if !c.control.now().Before(s.Proof.Deadline) || s.Proof.Deadline.Sub(c.control.now()) > 2*time.Hour {
		return hostedexecution.ErrRejected
	}
	if _, err := c.get(ctx, s); err == nil {
		return hostedexecution.ErrOwnership
	} else if err != hostedexecution.ErrNotFound {
		return err
	}
	if _, err := c.control.update(ctx, s.Proof.Reference, true, func(d *dispatch) error {
		if d.TimerCreateAttempted || d.DeadlineRemoving || d.DeadlineRemoved {
			return hostedexecution.ErrUnavailable
		}
		d.TimerProfile = c.config.profile()
		d.TimerCreateAttempted = true
		return nil
	}); err != nil {
		return err
	}
	step, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = c.workflows.CreateOrUpdate(step, c.config.ResourceGroup, c.config.name(s), c.config.definition(s), nil)
	err = timerError(err, true)
	if err == hostedexecution.ErrRejected {
		_, saveErr := c.control.update(ctx, s.Proof.Reference, false, func(d *dispatch) error { d.TimerCreateRejected = true; return nil })
		if saveErr != nil {
			return saveErr
		}
	}
	return err
}

func (c *SDKDeadlines) Verify(ctx context.Context, s DeadlineSpec) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, err := c.load(ctx, s)
	if err != nil {
		return err
	}
	if d.TimerCreateRejected {
		return hostedexecution.ErrRejected
	}
	if !d.TimerCreateAttempted || d.DeadlineRemoving || d.DeadlineRemoved {
		return hostedexecution.ErrUnavailable
	}
	actual, err := c.get(ctx, s)
	if err != nil {
		if err == hostedexecution.ErrNotFound {
			return hostedexecution.ErrUnavailable
		}
		return err
	}
	if err := c.config.verify(s, actual, true); err != nil {
		return err
	}
	_, err = c.control.update(ctx, s.Proof.Reference, false, func(d *dispatch) error {
		if d.DeadlineRemoving || d.DeadlineRemoved {
			return hostedexecution.ErrUnavailable
		}
		d.TimerSeen = true
		return nil
	})
	return err
}

var _ Deadlines = (*SDKDeadlines)(nil)
