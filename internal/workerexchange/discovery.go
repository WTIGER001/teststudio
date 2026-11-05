package workerexchange

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/suitespec"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func (s *Service) PrepareDiscovery(ctx context.Context, ref execution.Reference, input workerprotocol.DiscoveryInput, deadline time.Time) (Ticket, error) {
	if workerprotocol.ValidateDiscoveryInput(input) != nil || !domain.ValidID(ref.ID) || !domain.ValidID(ref.Installation) || !strings.HasPrefix(ref.Installation, "preparation-") || !domain.ValidID(ref.Run) || !domain.ValidID(ref.Worker) || ref.Test != "catalog" || ref.Attempt != 1 || input.Project != ref.Project || !strings.HasSuffix(ref.Profile, "-discovery-v1") {
		return Ticket{}, ErrInvalid
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return Ticket{}, ErrInvalid
	}
	return s.prepare(ctx, ref, payload, deadline, "discover", true, "")
}

func validateDiscoverySubmission(payload []byte, submission Submission) error {
	var input workerprotocol.DiscoveryInput
	if json.Unmarshal(payload, &input) != nil || workerprotocol.ValidateDiscoveryInput(input) != nil || submission.Discovery == nil || !reflect.DeepEqual(submission.Report, execution.CodedReport{}) || len(submission.Files) != 0 || submission.InfrastructureFailure != "" || submission.ProcessExitCode < -1 || submission.ProcessExitCode > 255 || len(submission.Discovery.Log) > 128<<10 {
		return ErrInvalid
	}
	report := submission.Discovery
	if report.Failure != "" {
		if len(report.Raw) != 0 || report.Failure != "worker" && report.Failure != "timeout" && report.Failure != "cancelled" {
			return ErrInvalid
		}
		return nil
	}
	if submission.ProcessExitCode != 0 || len(report.Raw) < 1 || len(report.Raw) > suitespec.DiscoveryLimit {
		return ErrInvalid
	}
	discovery, err := suitespec.DecodeDiscovery(report.Raw, input.Runner)
	if err != nil || discovery.Manifest.Runner != input.Runner || discovery.Manifest.Suite != input.Suite || discovery.Origin.SnapshotDigest != input.SourceDigest || !preparation.LocalSourceProfile(input.Runner, discovery.Manifest.BuildProfile) || discovery.Manifest.RequestPolicy != "" || discovery.Manifest.Browser == nil && len(discovery.Manifest.RequiredBindings) != 0 || discovery.Manifest.Browser != nil && (!input.Environment || discovery.Manifest.Browser.Requests != nil && !input.RequestPolicy) || suitespec.ValidateDiscovery(discovery, input.Image) != nil {
		return ErrInvalid
	}
	return nil
}

func (s *Service) DiscoveryObservation(ctx context.Context, ref execution.Reference, input workerprotocol.DiscoveryInput) (Submission, bool, error) {
	record, _, err := s.owned(ctx, ref)
	if err != nil {
		return Submission{}, false, err
	}
	payload, err := json.Marshal(input)
	if err != nil || record.Operation != "discover" || record.InputHash != fingerprint(payload) {
		return Submission{}, false, ErrConflict
	}
	if record.Submission == nil {
		return Submission{}, false, nil
	}
	if validateDiscoverySubmission(payload, *record.Submission) != nil {
		return Submission{}, false, ErrInvalid
	}
	data, _ := json.Marshal(record.Submission)
	var result Submission
	if json.Unmarshal(data, &result) != nil {
		return result, false, ErrUnavailable
	}
	return result, true, nil
}
