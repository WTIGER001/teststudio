package registryimage

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/adapters/registry/oci"
	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/preparation"
	"github.com/wtiger001/teststudio/internal/suitespec"
)

type Builder struct {
	Registry     *oci.Client
	RuntimeImage string
	Directory    string
}

func validReference(ref execution.Reference) bool {
	return domain.ValidID(ref.ID) && domain.ValidID(ref.Project) && domain.ValidID(ref.Installation) && strings.HasPrefix(ref.Installation, "preparation-")
}

func (b *Builder) Build(ctx context.Context, input preparation.BuildRequest) (result preparation.Build, err error) {
	result.StartedAt = time.Now().UTC()
	defer func() { result.FinishedAt = time.Now().UTC() }()
	if b.Registry == nil || !validReference(input.Reference) || suitespec.ValidateOrigin(input.Origin) != nil || input.Origin.SnapshotDigest != input.Snapshot.Digest {
		return result, errors.New("registry source assembly requires current ownership and matching source provenance")
	}
	manifest := input.Snapshot.Manifest
	if suitespec.Validate(manifest) != nil || suitespec.ValidateCatalog(manifest, input.Snapshot.Catalog) != nil || manifest.RequestPolicy != "" || manifest.Browser == nil && len(manifest.RequiredBindings) != 0 || !preparation.LocalSourceProfile(manifest.Runner, manifest.BuildProfile) {
		return result, errors.New("registry source assembly requires a validated Go or Playwright source profile")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	journal, parent, err := b.reserve(input)
	if err != nil {
		return result, err
	}
	defer parent.Close()
	defer journal.root.Close()
	defer func() {
		if err != nil {
			cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer stop()
			if cleanupErr := b.CleanupBuild(cleanup, input.Reference); cleanupErr != nil {
				err = errors.Join(err, preparation.ErrCleanupIncomplete)
			}
		}
	}()
	runtime, err := b.Registry.SourceRuntime(ctx, b.RuntimeImage)
	if err != nil {
		return result, err
	}
	if runtime.Runner != manifest.Runner {
		return result, errors.New("source runner differs from the trusted registry runtime")
	}
	if manifest.Browser != nil && (runtime.GatewayTransport != browsergateway.RemoteProtocol || runtime.SupervisorRevision == "" || manifest.Browser.Requests != nil && runtime.APIPolicy != "request-policy/v1") {
		return result, errors.New("environment source assembly requires an approved gateway-capable runtime and matching request-policy support")
	}
	if runtime.Runner == "playwright" {
		if err := preparation.ValidateBrowserDependencies(input.Snapshot, runtime.PlaywrightVersion); err != nil {
			return result, err
		}
	}
	layer, diff, err := sourceLayer(ctx, input.Snapshot, input.Origin)
	if err != nil {
		return result, err
	}
	metadata, err := b.Registry.PublishSource(ctx, runtime, oci.SourceLayer{Reference: input.Reference, SourceDigest: input.Snapshot.Digest, DiffID: diff, Compressed: layer}, journal)
	if err != nil {
		return result, err
	}
	result.Image, result.SourceDigest = metadata.Image.Reference, metadata.Source
	result.Log = "Source assembled and published by immutable distribution digest. Discovery and catalog publication remain required.\n"
	return result, nil
}

func (b *Builder) CleanupBuild(ctx context.Context, ref execution.Reference) error {
	if b.Registry == nil || !validReference(ref) {
		return errors.New("invalid registry assembly cleanup ownership")
	}
	parent, err := b.openDirectory(false)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer parent.Close()
	journal, err := readJournal(parent, ref)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer journal.root.Close()
	if journal.state.Opening != "" && journal.state.Upload == "" {
		return preparation.ErrCleanupIncomplete
	}
	if journal.state.Upload != "" {
		if err := b.Registry.CancelSourceUpload(ctx, journal.state.Runtime, journal.state.Upload); err != nil {
			return errors.Join(err, preparation.ErrCleanupIncomplete)
		}
	}
	return parent.RemoveAll(journalName(ref))
}

func (b *Builder) CompleteAssembly(ctx context.Context, ref execution.Reference) error {
	return b.CleanupBuild(ctx, ref)
}

var _ preparation.AssemblyBuilder = (*Builder)(nil)
