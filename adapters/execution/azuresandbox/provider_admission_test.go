package azuresandbox

import (
	"context"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

type admissionProbe struct {
	Admission
	after func()
}

func (a admissionProbe) VerifyInstance(ctx context.Context, c Config, s Snapshot) error {
	if err := a.Admission.VerifyInstance(ctx, c, s); err != nil {
		return err
	}
	a.after()
	return nil
}

func TestSandboxControllerRetriesAdmissionObservationAfterConcurrentWorkUpdate(t *testing.T) {
	for _, revoked := range []bool{false, true} {
		t.Run(map[bool]string{false: "healthy-work", true: "revoked-work"}[revoked], func(t *testing.T) {
			f := newControllerFixture(t, "go", "")
			ref := f.plan.proof.Reference
			if err := f.controller.Ensure(t.Context(), f.plan); err != nil {
				t.Fatal(err)
			}
			armed, changed := false, false
			f.controller.admission = admissionProbe{Admission: f.controls, after: func() { armed = true }}
			f.controller.now = func() time.Time {
				if armed && !changed {
					changed = true
					row, err := f.store.Get(t.Context(), workerexchange.Bucket, ref.ID)
					if err != nil {
						t.Fatal(err)
					}
					work, err := storage.Decode[workerexchange.Record](row)
					if err != nil {
						t.Fatal(err)
					}
					work.Revoked = revoked
					change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, work)
					if err := f.store.Commit(t.Context(), change); err != nil {
						t.Fatal(err)
					}
				}
				return time.Now()
			}
			state, err := f.controller.Observe(t.Context(), ref)
			if err != nil || !changed || state.AdmissionPending != revoked {
				t.Fatal("concurrent work update hid readiness or revocation", state, err)
			}
		})
	}
}
