package hostedexecution

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/suitespec"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

type DiscoveryExchange interface {
	Exchange
	PrepareDiscovery(context.Context, execution.Reference, workerprotocol.DiscoveryInput, time.Time) (workerexchange.Ticket, error)
	DiscoveryObservation(context.Context, execution.Reference, workerprotocol.DiscoveryInput) (workerprotocol.Submission, bool, error)
}

type DiscoveryInspector struct {
	Provider       Provider
	Exchange       DiscoveryExchange
	Name           string
	Runner         string
	ExchangeOrigin string
	PollInterval   time.Duration
}

func (i *DiscoveryInspector) DeferredCleanup() bool { return true }

func (i *DiscoveryInspector) Reference(ref execution.Reference) execution.Reference {
	ref = (&Executor{Name: i.Name, Runner: i.Runner}).Reference(ref)
	ref.Profile = i.Runner + "-" + i.Name + "-discovery-v1"
	return ref
}

func (i *DiscoveryInspector) valid(ref execution.Reference) bool {
	origin, err := url.Parse(i.ExchangeOrigin)
	return err == nil && origin.Scheme == "https" && origin.Hostname() != "" && origin.User == nil && origin.Path == "" && origin.RawQuery == "" && !origin.ForceQuery && origin.Fragment == "" && i.Provider != nil && i.Exchange != nil && name.MatchString(i.Name) && (i.Runner == "go" || i.Runner == "playwright") && i.PollInterval >= 0 && i.PollInterval <= 10*time.Second && domain.ValidID(ref.ID) && domain.ValidID(ref.Project) && domain.ValidID(ref.Installation) && strings.HasPrefix(ref.Installation, "preparation-") && domain.ValidID(ref.Run) && domain.ValidID(ref.Worker) && ref.Test == "catalog" && ref.Attempt == 1 && ref == i.Reference(ref)
}

func (i *DiscoveryInspector) controller() *Executor {
	return &Executor{Provider: i.Provider, Exchange: i.Exchange, PollInterval: i.PollInterval}
}

func (i *DiscoveryInspector) Inspect(ctx context.Context, request preparation.InspectionRequest) (result preparation.Inspection, resultErr error) {
	result.Image, result.StartedAt = request.Image, time.Now().UTC()
	defer func() { result.FinishedAt = time.Now().UTC() }()
	if !i.valid(request.Reference) || !domain.ValidID(request.Suite) || !distribution.MatchString(request.Image) {
		return result, ErrConfiguration
	}
	ctx, cancelPreparation := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelPreparation()
	var image Image
	err := bounded(ctx, func(ctx context.Context) error {
		var err error
		image, err = i.Provider.InspectImage(ctx, request.Image)
		return err
	})
	if err != nil {
		return result, ErrUnavailable
	}
	input := workerprotocol.DiscoveryInput{Project: request.Reference.Project, Suite: request.Suite, Runner: i.Runner, Image: request.Image, SourceDigest: image.SourceDigest}
	if image.Reference != request.Image || image.Project != input.Project || image.Runner != i.Runner || !runtime.MatchString(image.Runtime) || image.Supervisor != "worker-exchange/v1" || image.Discovery != "worker-exchange/v1" || image.OS != "linux" || image.Architecture != "arm64" && image.Architecture != "amd64" || workerprotocol.ValidateDiscoveryInput(input) != nil {
		return result, ErrOwnership
	}
	result.Project, result.Runtime = image.Project, image.Runtime
	controller := i.controller()
	deadline := time.Now().UTC().Add(2 * time.Minute)
	if at, ok := ctx.Deadline(); ok && at.Before(deadline) {
		deadline = at
	}
	ticket, err := i.Exchange.PrepareDiscovery(ctx, request.Reference, input, deadline)
	if err != nil {
		return result, ErrUnavailable
	}
	defer func() {
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := controller.revoke(finish, request.Reference); err != nil {
			resultErr = errors.Join(resultErr, preparation.ErrCleanupIncomplete)
		}
	}()
	if ticket.Execution != request.Reference.ID || len(ticket.Token) != 43 || !ticket.Deadline.After(time.Now()) || ticket.Deadline.After(deadline) {
		return result, ErrOwnership
	}
	wall, cancel := context.WithDeadline(ctx, ticket.Deadline)
	defer cancel()
	job := Job{Operation: "discover", Reference: request.Reference, Image: request.Image, Runner: i.Runner, ExchangeOrigin: i.ExchangeOrigin, Ticket: ticket}
	if err := bounded(wall, func(ctx context.Context) error { return i.Provider.Ensure(ctx, job) }); err != nil {
		return result, errors.Join(ErrUnavailable, preparation.ErrCleanupIncomplete)
	}
	status, err := controller.observe(wall, request.Reference, request.Image, ticket.Deadline)
	if err == nil && !terminal(status) {
		err = bounded(wall, func(ctx context.Context) error { return i.Exchange.Activate(ctx, request.Reference) })
		if err == nil {
			status, err = controller.waitWorker(wall, request.Reference, request.Image, ticket.Deadline, nil)
		}
	}
	if err != nil {
		return result, errors.Join(ErrUnavailable, preparation.ErrCleanupIncomplete)
	}
	submission, found, err := i.Exchange.DiscoveryObservation(wall, request.Reference, input)
	if err != nil || !found || !terminal(status) {
		return result, errors.Join(ErrUnavailable, preparation.ErrCleanupIncomplete)
	}
	result.Log, result.LogTruncated = submission.Discovery.Log, submission.Discovery.LogTruncated
	if submission.ProcessExitCode != 0 || submission.Discovery.Failure != "" {
		return result, errors.New("hosted source discovery failed; inspect the retained preparation log")
	}
	if status.Phase != "succeeded" || status.ExitCode == nil || *status.ExitCode != 0 {
		return result, errors.New("hosted source discovery did not finish successfully")
	}
	result.Discovery, err = suitespec.DecodeDiscovery(submission.Discovery.Raw, i.Runner)
	if err != nil || result.Discovery.Origin.SnapshotDigest != input.SourceDigest || suitespec.ValidateDiscovery(result.Discovery, request.Image) != nil {
		return result, ErrOwnership
	}
	return result, nil
}

func (i *DiscoveryInspector) CleanupInspection(ctx context.Context, request preparation.InspectionRequest) error {
	if !i.valid(request.Reference) {
		return ErrOwnership
	}
	state, err := i.Exchange.State(ctx, request.Reference)
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil || i.Exchange.CheckCleanup(ctx, request.Reference) != nil {
		return preparation.ErrCleanupIncomplete
	}
	controller := i.controller()
	if err := controller.revoke(ctx, request.Reference); err != nil {
		return preparation.ErrCleanupIncomplete
	}
	if _, err := controller.quiesce(ctx, request.Reference, request.Image, state.Deadline); err != nil && !errors.Is(err, ErrNotFound) {
		return preparation.ErrCleanupIncomplete
	}
	if err := i.Provider.Remove(ctx, request.Reference); err != nil && !errors.Is(err, ErrNotFound) {
		return preparation.ErrCleanupIncomplete
	}
	if err := i.Exchange.Cleanup(ctx, request.Reference); err != nil {
		return preparation.ErrCleanupIncomplete
	}
	return nil
}
