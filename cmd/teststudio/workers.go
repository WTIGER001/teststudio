package main

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/cliauth"
	"github.com/wtiger001/teststudio/internal/evidence"
	"github.com/wtiger001/teststudio/internal/notifications"
	"github.com/wtiger001/teststudio/internal/recording"
	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/schedules"
	"github.com/wtiger001/teststudio/internal/sources"
)

type backgroundServices struct {
	DisableExecution bool
	Runs             *runs.Service
	Recordings       *recording.Service
	CLI              *cliauth.Service
	Evidence         *evidence.Service
	Sources          *sources.Service
	Schedules        *schedules.Service
	Notifications    *notifications.Service
}

func (s backgroundServices) start(ctx context.Context) func() {
	ctx, cancel := context.WithCancel(ctx)
	done := []<-chan struct{}{
		s.Recordings.StartMaintenance(ctx),
		s.CLI.StartMaintenance(ctx),
		s.Evidence.StartMaintenance(ctx, 5*time.Second),
		s.Sources.StartMaintenance(ctx, 5*time.Second),
		s.Sources.StartPreparation(ctx),
	}
	if !s.DisableExecution {
		done = append(done, s.Runs.Start(ctx))
	}
	if s.Schedules != nil {
		done = append(done, s.Schedules.Start(ctx))
	}
	if s.Notifications != nil {
		done = append(done, s.Notifications.Start(ctx))
	}
	return func() {
		cancel()
		for _, finished := range done {
			if finished != nil {
				<-finished
			}
		}
	}
}
