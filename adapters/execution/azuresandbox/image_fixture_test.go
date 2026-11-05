package azuresandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func fixtureImagePlan(t *testing.T, p *Plan) *ImagePlan {
	t.Helper()
	c := p.config
	identity := "/subscriptions/" + c.Subscription + "/resourceGroups/studio-identities/providers/Microsoft.ManagedIdentity/userAssignedIdentities/studio-pull"
	image := hostedexecution.Image{Reference: p.proof.Image, Runner: p.proof.Runner, Runtime: "runtime-1", OS: "linux", Architecture: "amd64", Supervisor: "worker-exchange/v1", Sandbox: hostedexecution.SandboxSupervisor, Discovery: "worker-exchange/v1", Project: p.proof.Reference.Project, SourceDigest: p.proof.SourceDigest, TaskStorage: hostedexecution.NonRootTaskStorage}
	plan, err := NewImagePlan(c, identity, image, hash(p.proof.Reference))
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func imageSnapshot(t *testing.T, p *ImagePlan, id, state string) DiskImage {
	t.Helper()
	body := p.definition()
	body["id"] = id
	body["status"] = map[string]string{"state": state}
	data, _ := json.Marshal(body)
	image, err := (&Client{config: p.config}).image(data)
	if err != nil {
		t.Fatal(err)
	}
	return image
}

type imageControl struct {
	t               *testing.T
	mu              sync.Mutex
	items           map[string]DiskImage
	imports         int
	lost            bool
	empty           bool
	reject          bool
	inventoryErr    error
	beforeImport    func()
	beforeInventory func()
}

func (a *imageControl) ImportImage(_ context.Context, p *ImagePlan) (DiskImage, error) {
	if a.beforeImport != nil {
		a.beforeImport()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.imports++
	if a.reject {
		return DiskImage{}, hostedexecution.ErrRejected
	}
	if a.empty {
		return DiskImage{}, hostedexecution.ErrUnavailable
	}
	image := imageSnapshot(a.t, p, fmt.Sprintf("disk-%d", a.imports), "Building")
	a.items[image.ID] = image
	if a.lost {
		return DiskImage{}, hostedexecution.ErrUnavailable
	}
	return image, nil
}

func (a *imageControl) ReadImage(_ context.Context, id string) (DiskImage, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	image, found := a.items[id]
	if !found {
		return image, hostedexecution.ErrNotFound
	}
	return image, nil
}

func (a *imageControl) Images(context.Context) ([]DiskImage, error) {
	if a.beforeInventory != nil {
		a.beforeInventory()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.inventoryErr != nil {
		return nil, a.inventoryErr
	}
	items := []DiskImage{}
	for _, image := range a.items {
		items = append(items, image)
	}
	return items, nil
}

type imageAdmission struct {
	before func()
	err    error
}

func (a imageAdmission) BeforeImport(context.Context, Config, string) error {
	if a.before != nil {
		a.before()
	}
	return a.err
}

func newImageFixture(t *testing.T, runner string) (*controllerFixture, *ImageManager, *imageControl, hostedexecution.Job, *ImagePlan) {
	t.Helper()
	f := newControllerFixture(t, runner, "discover")
	p := fixtureImagePlan(t, f.plan)
	a := &imageControl{t: t, items: map[string]DiskImage{}}
	m, err := NewImageManager(ImageManagerOptions{Config: p.config, PullIdentity: p.proof.PullIdentity, Images: a, Exchange: f.exchange, Admission: imageAdmission{}, AllowImport: true})
	if err != nil {
		t.Fatal(err)
	}
	job := hostedexecution.Job{Reference: f.plan.proof.Reference, Image: p.proof.Image.Reference, Runner: runner, Operation: "discover", ExchangeOrigin: p.config.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: f.plan.proof.Reference.ID, Token: f.plan.token, Deadline: f.plan.proof.Deadline}}
	return f, m, a, job, p
}
