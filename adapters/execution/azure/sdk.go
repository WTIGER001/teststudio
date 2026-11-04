package azureworker

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v4"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type SDKClient struct {
	config     Config
	jobs       *a.JobsClient
	executions *a.JobsExecutionsClient
	http       *sdkHTTP
}

func NewSDKClient(c Config, credential azcore.TokenCredential, transport policy.Transporter) (*SDKClient, error) {
	if Validate(c) != nil || credential == nil {
		return nil, hostedexecution.ErrConfiguration
	}
	httpClient := newSDKHTTP(c, transport)
	options := &arm.ClientOptions{DisableRPRegistration: true, ClientOptions: policy.ClientOptions{
		Transport: httpClient, Retry: policy.RetryOptions{MaxRetries: -1},
		Cloud: cloud.Configuration{ActiveDirectoryAuthorityHost: "https://login.microsoftonline.com/", Services: map[cloud.ServiceName]cloud.ServiceConfiguration{cloud.ResourceManager: {Endpoint: "https://management.azure.com/", Audience: "https://management.core.windows.net/"}}},
	}}
	jobs, err := a.NewJobsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	executions, err := a.NewJobsExecutionsClient(c.Subscription, credential, options)
	if err != nil {
		return nil, hostedexecution.ErrConfiguration
	}
	return &SDKClient{config: c, jobs: jobs, executions: executions, http: httpClient}, nil
}

func (c *SDKClient) Close() { c.http.CloseIdleConnections() }

func sdkError(err error, mutation bool) error {
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
			case 400, 401, 403, 404, 422:
				return hostedexecution.ErrRejected
			}
		}
	}
	return controlError(err)
}

func (c *SDKClient) Get(ctx context.Context, name string) (a.Job, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := c.jobs.Get(ctx, c.config.ResourceGroup, name, nil)
	return result.Job, sdkError(err, false)
}

func (c *SDKClient) Create(ctx context.Context, name string, job a.Job) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := c.jobs.BeginCreateOrUpdate(ctx, c.config.ResourceGroup, name, job, nil)
	return sdkError(err, true)
}

func (c *SDKClient) Start(ctx context.Context, name string, options *a.JobsClientBeginStartOptions) (*a.JobExecutionBase, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if options == nil || options.ResumeToken != "" || options.Template == nil {
		return nil, hostedexecution.ErrOwnership
	}
	poller, err := c.jobs.BeginStart(ctx, c.config.ResourceGroup, name, options)
	if err != nil {
		return nil, sdkError(err, true)
	}
	if !poller.Done() {
		return nil, nil
	}
	result, err := poller.Result(ctx)
	return &result.JobExecutionBase, sdkError(err, false)
}

func (c *SDKClient) Executions(ctx context.Context, name string) ([]a.JobExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pager := c.executions.NewListPager(c.config.ResourceGroup, name, nil)
	items := []a.JobExecution{}
	seen := map[string]bool{}
	for pages := 0; pager.More() && pages < 4; pages++ {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, sdkError(err, false)
		}
		if len(page.Value) > 100 {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, item := range page.Value {
			if item == nil {
				return nil, hostedexecution.ErrOwnership
			}
			items = append(items, *item)
			if len(items) > 1 {
				return items, nil
			}
		}
		if next := value(page.NextLink); next != "" {
			u, err := url.Parse(next)
			if err != nil || u.Scheme != "https" || u.Host != "management.azure.com" || u.Path != c.config.resourceID("Microsoft.App", "jobs", name)+"/executions" || u.User != nil || u.Fragment != "" || u.Query().Get("api-version") != "2025-07-01" || seen[next] {
				return nil, hostedexecution.ErrOwnership
			}
			seen[next] = true
		}
	}
	if pager.More() {
		return nil, hostedexecution.ErrUnavailable
	}
	return items, nil
}

func (c *SDKClient) Stop(ctx context.Context, name, execution string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := c.jobs.BeginStopExecution(ctx, c.config.ResourceGroup, name, execution, nil)
	return sdkError(err, false)
}

func (c *SDKClient) Delete(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := c.jobs.BeginDelete(ctx, c.config.ResourceGroup, name, nil)
	return sdkError(err, false)
}

var _ Jobs = (*SDKClient)(nil)
