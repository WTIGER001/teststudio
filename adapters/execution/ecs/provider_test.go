package ecsworker

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func TestECSProviderKeepsAdmissionAndCheckpointedCleanupSeparate(t *testing.T) {
	for _, runner := range []string{"go", "playwright"} {
		for _, operation := range []string{"", "discover"} {
			t.Run(runner+operation, func(t *testing.T) {
				p, f, job := lifecycle(t, runner, operation, "")
				if err := p.Ensure(t.Context(), job); err != nil {
					t.Fatal(err)
				}
				f.networkPending = true
				status, err := p.Observe(t.Context(), job.Reference)
				if err != nil || !status.AdmissionPending {
					t.Fatal("unverified network opened admission", status, err)
				}
				f.networkPending = false
				status, err = p.Observe(t.Context(), job.Reference)
				if err != nil || status.AdmissionPending {
					t.Fatal(status, err)
				}
				if err := p.Remove(t.Context(), job.Reference); err == nil {
					t.Fatal("resources deleted before checkpoint")
				}
				revoke(t, p, job, false)
				stopObserved(t, p, f, job)
				f.task = nil
				if _, err := p.Observe(t.Context(), job.Reference); err != nil {
					t.Fatal("terminal receipt disappeared with task history", err)
				}
				if err := p.Remove(t.Context(), job.Reference); err == nil {
					t.Fatal("ticket revocation replaced the checkpoint")
				}
				revoke(t, p, job, true)
				f.checkpointed = true
				if err := p.Remove(t.Context(), job.Reference); err != nil {
					t.Fatal(err)
				}
				if err := p.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrRejected) {
					t.Fatal("removed task was not fenced", err)
				}
				row, _ := p.store.Get(t.Context(), Bucket, job.Reference.ID)
				if strings.Contains(string(row.Data), job.Ticket.Token) || f.registrations != 1 || f.launches != 1 || f.deletes != 1 {
					t.Fatal("dispatch journal leaked or repeated work")
				}
			})
		}
	}
}

func TestECSLostAcknowledgementsRecoverWithoutCreatingMoreResources(t *testing.T) {
	for _, mode := range []string{"lost-register", "lost-launch", "lost-deadline", "launch-rejected"} {
		t.Run(mode, func(t *testing.T) {
			p, f, job := lifecycle(t, "go", "", mode)
			if err := p.Ensure(t.Context(), job); err == nil {
				t.Fatal("fixture did not lose its acknowledgement")
			}
			p = reopenProvider(t, p, f)
			revoke(t, p, job, false)
			if mode == "lost-launch" || mode == "lost-deadline" {
				stopObserved(t, p, f, job)
			}
			revoke(t, p, job, true)
			if err := p.Remove(t.Context(), job.Reference); err != nil {
				t.Fatal(err)
			}
			if f.registrations != 1 || f.launches > 1 || f.schedules > 1 || f.deletes != 1 {
				t.Fatal("recovery repeated provider creation")
			}
		})
	}
}

func TestECSAdmissionRejectsBeforeCreationAndKeepsPrivateErrorsInternal(t *testing.T) {
	p, f, job := lifecycle(t, "go", "", "")
	f.networkRejected = true
	if err := p.Ensure(t.Context(), job); !errors.Is(err, hostedexecution.ErrRejected) || f.registrations != 0 || f.launches != 0 {
		t.Fatal("rejected pool created provider resources", err)
	}
	p, f, job = lifecycle(t, "go", "", "")
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	f.mode = "read-outage"
	if _, err := p.Observe(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrUnavailable) || strings.Contains(err.Error(), "canary") {
		t.Fatal("provider error escaped into public diagnostics", err)
	}
}

func TestECSUnknownLaunchAndMissingLiveHistoryRemainUnresolved(t *testing.T) {
	p, f, job := lifecycle(t, "go", "", "hidden-launch")
	if err := p.Ensure(t.Context(), job); err == nil {
		t.Fatal("unknown launch succeeded")
	}
	revoke(t, p, job, true)
	if err := p.Remove(t.Context(), job.Reference); err == nil || f.deletes != 0 {
		t.Fatal("empty inventory was treated as a stopped task")
	}
	f.makeTask()
	stopObserved(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); err != nil || f.launches != 1 {
		t.Fatal("late task could not reconcile without another launch", err)
	}

	p, f, job = lifecycle(t, "playwright", "", "")
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	f.task = nil
	if _, err := p.Observe(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("missing live task was considered terminal", err)
	}
	revoke(t, p, job, true)
	if err := p.Remove(t.Context(), job.Reference); err == nil || f.deletes != 0 {
		t.Fatal("missing live task permitted cleanup")
	}
}

func TestECSDelayedDefinitionDeletionBacksOffAndKeepsTheResult(t *testing.T) {
	p, f, job := lifecycle(t, "go", "", "deletion-pending")
	if err := p.Ensure(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	revoke(t, p, job, true)
	stopObserved(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); !errors.Is(err, hostedexecution.ErrUnavailable) {
		t.Fatal("deletion was treated as completed", err)
	}
	reads := f.reads
	if err := p.Remove(t.Context(), job.Reference); err == nil || f.reads != reads {
		t.Fatal("pending deletion did not back off")
	}
	f.now = f.now.Add(61 * time.Second)
	f.definition = nil
	if err := p.Remove(t.Context(), job.Reference); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Observe(t.Context(), job.Reference); err != nil {
		t.Fatal("deletion discarded the saved result", err)
	}
}

func TestECSConcurrentControllersReserveOnlyOneRegistrationAndLaunch(t *testing.T) {
	p, f, job := lifecycle(t, "go", "", "")
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() { _ = p.Ensure(t.Context(), job) })
	}
	group.Wait()
	if f.registrations != 1 || f.launches != 1 || f.schedules != 1 {
		t.Fatal("concurrent controllers duplicated resource creation")
	}
	refs, err := p.Inventory(t.Context(), job.Reference.Installation, job.Reference.Profile)
	if err != nil || len(refs) != 1 || refs[0] != job.Reference {
		t.Fatal(refs, err)
	}
	actual := copyValue(f.schedule)
	actual.Target.RoleArn = aws.String(p.config.ExecutionRole)
	f.schedule = actual
	revoke(t, p, job, true)
	stopObserved(t, p, f, job)
	if err := p.Remove(t.Context(), job.Reference); err == nil || f.deletes != 0 {
		t.Fatal("changed stop schedule was deleted as owned")
	}
}
