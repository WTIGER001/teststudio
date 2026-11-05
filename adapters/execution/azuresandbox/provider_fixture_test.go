package azuresandbox

import (
	"context"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

type inspectCall func(context.Context, string) (hostedexecution.Image, error)

func (f inspectCall) InspectImage(ctx context.Context, image string) (hostedexecution.Image, error) {
	return f(ctx, image)
}

func fixtureProvider(t *testing.T, f *controllerFixture, m *ImageManager, meta hostedexecution.Image) *Provider {
	t.Helper()
	mode := "execution"
	if f.plan.proof.Operation == "discover" {
		mode = "discovery"
	}
	p, err := NewProvider(ProviderOptions{Mode: mode, Controller: f.controller, Images: m, Inspector: inspectCall(func(_ context.Context, ref string) (hostedexecution.Image, error) {
		if ref != meta.Reference {
			return hostedexecution.Image{}, hostedexecution.ErrOwnership
		}
		return meta, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func preparedImage(t *testing.T, f *controllerFixture, a *imageControl, meta hostedexecution.Image) {
	t.Helper()
	ref := f.plan.proof.Reference
	ref.ID = "prepared-source"
	ref.Installation = "preparation-studio"
	ref.Test = "catalog"
	ref = (&hostedexecution.DiscoveryInspector{Name: Location, Runner: meta.Runner}).Reference(ref)
	identity := fixtureImagePlan(t, f.plan).proof.PullIdentity
	plan, err := NewImagePlan(f.plan.config, identity, meta, hash(ref))
	if err != nil {
		t.Fatal(err)
	}
	image := imageSnapshot(t, plan, "prepared-disk", "Ready")
	a.items[image.ID] = image
	d := imageImport{Proof: plan.proof, RequestedBy: ref, CreatedAt: time.Now().UTC(), Attempted: true, Receipt: image.ID, LastState: "Ready"}
	change, err := storage.Put(ImageBucket, plan.key(), meta.Project, 0, d)
	if err != nil || f.store.Commit(t.Context(), change) != nil {
		t.Fatal("source preparation receipt is unavailable", err)
	}
}
