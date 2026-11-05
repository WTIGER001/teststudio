package azuresandbox

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func TestSandboxImageImportPersistsReadyLookupAcrossRestart(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		t.Run(runner, func(t *testing.T) {
			f, m, a, job, plan := newImageFixture(t, runner)
			image, err := m.Prepare(t.Context(), job, plan.proof.Image)
			if err != nil || image.State != "Building" {
				t.Fatal("conversion did not remain pending", err)
			}
			a.mu.Lock()
			image.State = "Ready"
			a.items[image.ID] = image
			a.mu.Unlock()
			f.reopen()
			reader, err := NewImageManager(ImageManagerOptions{Config: m.config, PullIdentity: m.identity, Images: a, Exchange: &workerexchange.Service{Store: f.store}})
			if err != nil {
				t.Fatal(err)
			}
			ready, err := reader.Lookup(t.Context(), plan.proof.Image)
			if err != nil || ready.ID != image.ID || ready.State != "Ready" || a.imports != 1 {
				t.Fatal("ready conversion was lost or imported again", err)
			}
			if _, err := reader.Prepare(t.Context(), job, plan.proof.Image); err != hostedexecution.ErrConfiguration {
				t.Fatal("read-only process can import", err)
			}
			row, err := f.store.Get(t.Context(), ImageBucket, plan.key())
			if err != nil || strings.Contains(string(row.Data), job.Ticket.Token) || strings.Contains(string(row.Data), identityToken) {
				t.Fatal("import journal contains credentials", err)
			}
			a.mu.Lock()
			delete(a.items, image.ID)
			a.mu.Unlock()
			if _, err := m.Prepare(t.Context(), job, plan.proof.Image); err != hostedexecution.ErrUnavailable || a.imports != 1 {
				t.Fatal("missing acknowledged conversion was recreated", err)
			}
		})
	}
}

func TestSandboxImageImportReconcilesLostReceiptWithoutReplay(t *testing.T) {
	for _, mode := range []string{"one", "none", "duplicate", "incomplete"} {
		t.Run(mode, func(t *testing.T) {
			f, m, a, job, plan := newImageFixture(t, "go")
			a.lost = true
			if mode == "none" {
				a.empty = true
			}
			if _, err := m.Prepare(t.Context(), job, plan.proof.Image); err != hostedexecution.ErrUnavailable {
				t.Fatal(err)
			}
			f.reopen()
			if mode == "duplicate" {
				a.items["second-disk"] = imageSnapshot(t, plan, "second-disk", "Ready")
			}
			if mode == "incomplete" {
				a.inventoryErr = hostedexecution.ErrUnavailable
			}
			image, err := m.Lookup(t.Context(), plan.proof.Image)
			want := error(hostedexecution.ErrUnavailable)
			if mode == "one" {
				want = nil
			}
			if mode == "duplicate" {
				want = hostedexecution.ErrOwnership
			}
			if err != want || mode == "one" && image.ID != "disk-1" {
				t.Fatal("uncertain ownership was misclassified", image, err)
			}
			_, _ = m.Prepare(t.Context(), job, plan.proof.Image)
			if a.imports != 1 {
				t.Fatal("lost receipt repeated an import")
			}
		})
	}
}

func TestSandboxImageImportHasOneReservationAcrossReplicas(t *testing.T) {
	_, m, a, job, plan := newImageFixture(t, "playwright")
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, _ = m.Prepare(t.Context(), job, plan.proof.Image)
		}()
	}
	group.Wait()
	if a.imports != 1 {
		t.Fatal("concurrent preparation duplicated conversion", a.imports)
	}
	if _, err := m.Lookup(t.Context(), plan.proof.Image); err != nil {
		t.Fatal("winning receipt is unavailable", err)
	}
}

func TestSandboxImageImportRequiresCurrentDiscoveryAuthorization(t *testing.T) {
	for _, mode := range []string{"execution", "source", "ticket", "permission", "revoked", "admission", "foreign-project", "unreserved", "grant-revision"} {
		t.Run(mode, func(t *testing.T) {
			f, m, a, job, plan := newImageFixture(t, "go")
			switch mode {
			case "execution":
				job.Operation = ""
			case "source":
				plan.proof.Image.SourceDigest = "sha256:" + strings.Repeat("d", 64)
			case "ticket":
				job.Ticket.Token = secretTicket
			case "permission":
				a.beforeInventory = func() { f.denied.Store(true) }
			case "revoked":
				m.admission = imageAdmission{before: func() {
					if err := f.exchange.Revoke(t.Context(), job.Reference); err != nil {
						t.Fatal(err)
					}
				}}
			case "admission":
				m.admission = imageAdmission{err: hostedexecution.ErrUnavailable}
			case "foreign-project":
				job.Reference.Project = "unrelated-project"
			case "unreserved":
				a.items["orphan-disk"] = imageSnapshot(t, plan, "orphan-disk", "Ready")
			case "grant-revision":
				change, _ := storage.Put("sandbox-import-test-grant", "grant", job.Reference.Project, 0, true)
				if err := f.store.Commit(t.Context(), change); err != nil {
					t.Fatal(err)
				}
				grant, err := f.store.Get(t.Context(), change.Bucket, change.ID)
				if err != nil {
					t.Fatal(err)
				}
				calls := 0
				f.exchange.Authorize = func(ctx context.Context, _ execution.Reference) (workerstate.Access, error) {
					calls++
					if calls == 4 {
						revoke, _ := storage.Put(grant.Bucket, grant.ID, grant.Project, grant.Revision, false)
						if err := f.store.Commit(ctx, revoke); err != nil {
							t.Fatal(err)
						}
					}
					return workerstate.Access{Deadline: job.Ticket.Deadline, Records: []storage.Record{grant}}, nil
				}
			}
			if _, err := m.Prepare(t.Context(), job, plan.proof.Image); err == nil || a.imports != 0 {
				t.Fatal("unapproved import reached provider", err, a.imports)
			}
		})
	}
}

func nextImageRequest(t *testing.T, f *controllerFixture, job hostedexecution.Job) hostedexecution.Job {
	t.Helper()
	job.Reference.ID += "-next"
	job.Reference = (&hostedexecution.DiscoveryInspector{Name: Location, Runner: job.Runner}).Reference(job.Reference)
	var err error
	job.Ticket, err = f.exchange.PrepareDiscovery(t.Context(), job.Reference, workerprotocol.DiscoveryInput{Project: job.Reference.Project, Suite: "parcel-inspection", Runner: job.Runner, Image: job.Image, SourceDigest: f.plan.proof.SourceDigest}, job.Ticket.Deadline)
	if err != nil {
		t.Fatal(err)
	}
	return job
}

func TestSandboxImageImportRetriesOnlyKnownFailureForNewDiscovery(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		t.Run(map[bool]string{false: "failed-build", true: "rejected-import"}[rejected], func(t *testing.T) {
			f, m, a, job, plan := newImageFixture(t, "go")
			a.reject = rejected
			_, err := m.Prepare(t.Context(), job, plan.proof.Image)
			if rejected && err != hostedexecution.ErrRejected || !rejected && err != nil {
				t.Fatal(err)
			}
			if !rejected {
				a.items["disk-1"] = imageSnapshot(t, plan, "disk-1", "Failed")
				if _, err := m.Lookup(t.Context(), plan.proof.Image); err != hostedexecution.ErrRejected {
					t.Fatal(err)
				}
			}
			if _, err := m.Prepare(t.Context(), job, plan.proof.Image); err != hostedexecution.ErrRejected || a.imports != 1 {
				t.Fatal("same request repeated failed conversion", err)
			}
			f.reopen()
			a.reject = false
			next := nextImageRequest(t, f, job)
			image, err := m.Prepare(t.Context(), next, plan.proof.Image)
			if err != nil || image.ID != "disk-2" || a.imports != 2 {
				t.Fatal("new authorized request cannot retry known failure", err)
			}
			history, err := f.store.All(t.Context(), ImageHistoryBucket, job.Reference.Project)
			if err != nil || len(history) != 1 {
				t.Fatal("previous conversion receipt was lost", err)
			}
			old, err := storage.Decode[imageImport](history[0])
			if err != nil || old.RequestedBy != job.Reference || old.Proof.Fingerprint != plan.proof.Fingerprint || !failedImport(old) {
				t.Fatal("conversion history changed", err)
			}
			data, _ := json.Marshal(old)
			if strings.Contains(string(data), job.Ticket.Token) {
				t.Fatal("history contains ticket")
			}
			current, err := NewImagePlan(m.config, m.identity, plan.proof.Image, hash(next.Reference))
			if err != nil {
				t.Fatal(err)
			}
			a.items["disk-2"] = imageSnapshot(t, current, "disk-2", "Failed")
			if _, err := m.Lookup(t.Context(), plan.proof.Image); err != hostedexecution.ErrRejected {
				t.Fatal(err)
			}
			if _, err := m.Prepare(t.Context(), job, plan.proof.Image); err != hostedexecution.ErrRejected || a.imports != 2 {
				t.Fatal("archived discovery repeated conversion", err)
			}
		})
	}
}

func TestSandboxImageImportRejectsReceiptDrift(t *testing.T) {
	_, m, a, job, plan := newImageFixture(t, "go")
	image, err := m.Prepare(context.Background(), job, plan.proof.Image)
	if err != nil {
		t.Fatal(err)
	}
	image.DefinitionHash = hash("another source, project, labels, or supervisor")
	a.items[image.ID] = image
	if _, err := m.Lookup(t.Context(), plan.proof.Image); err != hostedexecution.ErrOwnership {
		t.Fatal("changed image was admitted", err)
	}
}
