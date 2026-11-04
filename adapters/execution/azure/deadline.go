package azureworker

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	l "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

type DeadlineOptions struct {
	Config           DeadlineConfig
	Store            *storage.Store
	Jobs             Jobs
	Credential       azcore.TokenCredential
	Transport        policy.Transporter
	DiscoveryCleanup func(context.Context, execution.Reference) error
	Now              func() time.Time
}

type SDKDeadlines struct {
	config    DeadlineConfig
	provider  *Provider
	workflows *l.WorkflowsClient
	http      *deadlineHTTP
}

func NewSDKDeadlines(o DeadlineOptions) (*SDKDeadlines, error) {
	if o.Config.validate() != nil || o.Store == nil || o.Jobs == nil || o.Credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	h := newDeadlineHTTP(o.Config, o.Transport)
	client, err := l.NewWorkflowsClient(o.Config.Worker.Subscription, o.Credential, deadlineClientOptions(h))
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKDeadlines{config: o.Config, provider: &Provider{config: o.Config.Worker, store: o.Store, jobs: o.Jobs, discoveryCleanup: o.DiscoveryCleanup, now: now}, workflows: client, http: h}, nil
}

func (c *SDKDeadlines) Close() { c.http.CloseIdleConnections() }

func (c *SDKDeadlines) load(ctx context.Context, s DeadlineSpec) (dispatch, error) {
	if c.config.validateSpec(s) != nil {
		return dispatch{}, hostedexecution.ErrOwnership
	}
	d, _, plan, err := c.provider.load(ctx, s.Reference)
	if err != nil {
		return d, controlError(err)
	}
	expected := plan.deadlineSpec()
	if !d.DeadlineAttempted || expected.ResourceID != s.ResourceID || expected.Specification != s.Specification || !expected.At.Equal(s.At) || d.DeadlineProfile != "" && d.DeadlineProfile != c.config.profile() {
		return d, hostedexecution.ErrOwnership
	}
	return d, nil
}

func (c *SDKDeadlines) Ensure(ctx context.Context, s DeadlineSpec) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, err := c.load(ctx, s)
	if err != nil {
		return err
	}
	if d.DeadlineRemoving || d.DeadlineRemoved || d.DeadlineCreateRejected {
		return hostedexecution.ErrRejected
	}
	if d.DeadlineCreateAttempted {
		return c.Verify(ctx, s)
	}
	if !c.provider.now().Before(s.At) || s.At.Sub(c.provider.now()) > 2*time.Hour {
		return hostedexecution.ErrRejected
	}
	if _, err := c.get(ctx, s); err == nil {
		return hostedexecution.ErrOwnership
	} else if !errors.Is(err, hostedexecution.ErrNotFound) {
		return err
	}
	if _, err := c.provider.update(ctx, s.Reference, true, func(d *dispatch) error {
		if d.DeadlineCreateAttempted || d.DeadlineRemoving || d.DeadlineRemoved || d.DeadlineCreateRejected {
			return hostedexecution.ErrUnavailable
		}
		d.DeadlineProfile, d.DeadlineCreateAttempted = c.config.profile(), true
		return nil
	}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = c.workflows.CreateOrUpdate(ctx, c.config.ResourceGroup, c.config.name(s), c.config.definition(s), nil)
	err = sdkError(err, true)
	if errors.Is(err, hostedexecution.ErrRejected) {
		_, saveErr := c.provider.update(ctx, s.Reference, false, func(d *dispatch) error { d.DeadlineCreateRejected = true; return nil })
		if saveErr != nil {
			return saveErr
		}
	}
	return err
}

func (c *SDKDeadlines) get(ctx context.Context, s DeadlineSpec) (l.Workflow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.workflows.Get(ctx, c.config.ResourceGroup, c.config.name(s), nil)
	return result.Workflow, sdkError(err, false)
}

func (c *SDKDeadlines) Verify(ctx context.Context, s DeadlineSpec) (resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	d, err := c.load(ctx, s)
	if err != nil {
		return err
	}
	if !d.DeadlineCreateAttempted || d.DeadlineCreateRejected || d.DeadlineRemoving || d.DeadlineRemoved {
		return hostedexecution.ErrUnavailable
	}
	actual, err := c.get(ctx, s)
	if err != nil {
		return err
	}
	if err := c.config.verify(s, actual, true); err != nil {
		return err
	}
	_, err = c.provider.update(ctx, s.Reference, false, func(d *dispatch) error {
		if d.DeadlineRemoving || d.DeadlineRemoved {
			return hostedexecution.ErrUnavailable
		}
		d.DeadlineSeen = true
		return nil
	})
	return err
}

var _ Deadlines = (*SDKDeadlines)(nil)
