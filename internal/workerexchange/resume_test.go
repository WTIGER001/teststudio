package workerexchange_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func TestResumeDispatchReauthorizesOnlyUnactivatedGatedWork(t *testing.T) {
	for _, mode := range []string{"restart", "activated", "revoked", "permission", "ungated", "expired", "guard-race"} {
		t.Run(mode, func(t *testing.T) {
			f := fresh(t)
			f.input.AwaitDispatch = mode != "ungated"
			ticket := f.prepare(t)
			s := &workerexchange.Service{Store: f.store, Authorize: f.service.Authorize}
			switch mode {
			case "activated":
				if err := f.service.Activate(t.Context(), f.ref); err != nil {
					t.Fatal(err)
				}
			case "revoked":
				if err := f.service.Revoke(t.Context(), f.ref); err != nil {
					t.Fatal(err)
				}
			case "permission":
				f.revokeAuthority(t)
			case "expired":
				row, err := f.store.Get(t.Context(), workerexchange.Bucket, f.ref.ID)
				if err != nil {
					t.Fatal(err)
				}
				r, _ := storage.Decode[workerexchange.Record](row)
				r.Deadline = time.Now().Add(-time.Second)
				change, _ := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, r)
				if err := f.store.Commit(t.Context(), change); err != nil {
					t.Fatal(err)
				}
			case "guard-race":
				s.Authorize = func(ctx context.Context, ref execution.Reference) (workerstate.Access, error) {
					a, err := f.service.Authorize(ctx, ref)
					if err == nil {
						f.revokeAuthority(t)
					}
					return a, err
				}
			}
			resumed, err := s.ResumeDispatch(t.Context(), f.ref)
			if mode == "restart" {
				if err != nil || resumed.Token != ticket.Token || resumed.Execution != ticket.Execution || !resumed.Deadline.Equal(ticket.Deadline) {
					t.Fatal("restart changed the prepared ticket", err)
				}
				if _, err := s.Input(t.Context(), f.ref.ID, resumed.Token, "worker"); !errors.Is(err, workerexchange.ErrUnavailable) {
					t.Fatal("resumption activated executable input", err)
				}
			} else if err == nil || resumed.Token != "" {
				t.Fatal("invalid authority recovered a usable ticket")
			}
		})
	}
}
