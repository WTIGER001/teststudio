package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"slices"

	"github.com/wtiger001/teststudio/internal/runs"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (s *Service) Prepare(ctx context.Context, run runs.Run) ([]storage.Change, error) {
	row, err := s.Store.Get(ctx, "notification-registry", run.Project)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	list, err := storage.Decode[registry](row)
	if err != nil {
		return nil, err
	}
	v := plan{Project: run.Project, Run: run.ID, Channels: []subscription{}}
	changes := []storage.Change{{Record: row, Assert: true}}
	for _, id := range list.IDs {
		channel, cRow, err := s.channel(ctx, run.Project, id)
		if err != nil {
			return nil, err
		}
		if !channel.Enabled || channel.Archived || !scopeMatches(channel.Definition, run) {
			continue
		}
		v.Channels = append(v.Channels, subscription{Channel: channel, Revision: cRow.Revision})
		changes = append(changes, storage.Change{Record: cRow, Assert: true})
	}
	if len(v.Channels) > 0 {
		c, err := storage.Put("notification-plans", run.ID, run.Project, 0, v)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, nil
}
func scopeMatches(d Definition, run runs.Run) bool {
	if d.Application != "" {
		app := run.Request.Application
		if run.Environment != nil {
			app = run.Environment.Application
		}
		if app == "" && len(run.Applications) == 1 {
			app = run.Applications[0]
		}
		if app != d.Application {
			return false
		}
		for _, test := range run.Tests {
			if test.Application != "" && test.Application != app {
				return false
			}
		}
	}
	stage := ""
	if run.Environment != nil {
		stage = run.Environment.Stage
	}
	for _, filter := range []struct {
		values []string
		value  string
	}{{d.Suites, run.Request.Suite}, {d.Environments, run.Request.Environment}, {d.Stages, stage}, {d.Triggers, run.Trigger}} {
		if len(filter.values) > 0 && !slices.Contains(filter.values, filter.value) {
			return false
		}
	}
	return true
}
func (s *Service) Complete(ctx context.Context, run runs.Run) ([]storage.Change, error) {
	row, err := s.Store.Get(ctx, "notification-plans", run.ID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v, err := storage.Decode[plan](row)
	if err != nil {
		return nil, err
	}
	if v.Project != run.Project || v.Run != run.ID || row.Project != run.Project {
		return nil, errors.New("notification plan ownership does not match its run")
	}
	changes := []storage.Change{}
	for _, subscription := range v.Channels {
		d := subscription.Channel.Definition
		if !slices.Contains(d.Outcomes, run.Outcome) && !(d.EvidenceGaps && run.EvidenceState != "complete") {
			continue
		}
		if run.FinishedAt == nil {
			return nil, errors.New("notification completion requires a finished run")
		}
		link, err := url.Parse(s.Origin)
		if err != nil || link.Host == "" || (link.Scheme != "http" && link.Scheme != "https") || link.User != nil || link.RawQuery != "" || link.Fragment != "" || link.Path != "" || len(s.Origin) > 512 {
			return nil, errors.New("notification UI origin is invalid")
		}
		link.Path = "/projects/" + run.Project + "/runs/" + run.ID
		sum := sha256.Sum256([]byte(run.ID + "/" + d.ID))
		id := run.FinishedAt.UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(sum[:])
		message := Message{Schema: "teststudio.notification/v1", Event: "run.finished", Delivery: id, Project: run.Project, Run: run.ID, Suite: run.Request.Suite, Environment: run.Request.Environment, Trigger: run.Trigger, State: run.State, Outcome: run.Outcome, Evidence: run.EvidenceState, Counts: run.Counts, FinishedAt: *run.FinishedAt, URL: link.String()}
		if run.Environment != nil {
			message.Stage = run.Environment.Stage
		}
		delivery := Delivery{ID: id, Project: run.Project, Run: run.ID, Channel: d.ID, ChannelName: d.Name, DispatchEpoch: subscription.Channel.Epoch, Destination: subscription, Message: message, State: "pending", AttemptLimit: d.MaxAttempts, NextAt: s.now(), CreatedAt: s.now(), UpdatedAt: s.now()}
		c, err := storage.Put(Deliveries(run.Project), id, run.Project, 0, delivery)
		if err != nil {
			return nil, err
		}
		j, err := storage.Put("notification-work", id, run.Project, 0, job{Project: run.Project, Delivery: id})
		if err != nil {
			return nil, err
		}
		changes = append(changes, c, j)
	}
	return changes, nil
}
