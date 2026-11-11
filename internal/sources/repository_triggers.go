package sources

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sourceprovider"
	"github.com/wtiger001/teststudio/internal/storage"
)

type repositoryTrigger struct {
	Revision   *sourceprovider.Revision
	Kind       string
	Generation int64
	Delivery   string
	Digest     string
	Timer      *storage.Record
	Due        time.Time
	Now        time.Time
}
type repositoryDelivery struct {
	Sync   string `json:"syncId"`
	Digest string `json:"digest"`
}
type WebhookReceipt struct {
	State string `json:"state"`
	Sync  string `json:"syncId,omitempty"`
}

func repositoryDeliveries(project, source string) string {
	return "repository-deliveries/" + project + "/" + source
}

func (s *Service) ReceiveRepositoryWebhook(ctx context.Context, project, source string, request sourceprovider.WebhookRequest) (WebhookReceipt, error) {
	reject := sourceprovider.ErrWebhookRejected
	if !domain.ValidID(project) || !domain.ValidID(source) {
		return WebhookReceipt{}, reject
	}
	row, err := s.Store.Get(ctx, repositoryBucket(project), source)
	if err != nil {
		return WebhookReceipt{}, reject
	}
	registered, err := storage.Decode[RepositorySource](row)
	if err != nil {
		return WebhookReceipt{}, reject
	}
	config := registered.Configuration
	if request.Provider != "" && request.Provider != config.Provider {
		return WebhookReceipt{}, reject
	}
	verifier, ok := s.Providers[config.Provider].(sourceprovider.WebhookVerifier)
	if !ok || !config.Enabled || !config.Push || registered.RepositoryID == "" {
		return WebhookReceipt{}, reject
	}
	access := sourceprovider.Access{Project: project, Repository: config.Repository, RepositoryID: registered.RepositoryID, Credential: config.Credential}
	event, err := verifier.VerifyWebhook(ctx, access, config.WebhookCredential, request)
	if err != nil || event.Delivery != request.Delivery || event.RepositoryID != registered.RepositoryID || !strings.EqualFold(event.Repository, config.Repository) {
		return WebhookReceipt{}, reject
	}
	if event.Kind != "push" || event.Reference != config.Reference || event.Deleted {
		return WebhookReceipt{State: "ignored"}, nil
	}
	queued, err := s.enqueueRepositorySync(ctx, registered.Actor, project, source, "webhook/"+event.Delivery, SyncInput{}, repositoryTrigger{Kind: "push", Generation: registered.Generation, Delivery: event.Delivery, Digest: digest(request.Event + "\x00" + string(request.Body))})
	if err != nil {
		return WebhookReceipt{}, err
	}
	return WebhookReceipt{State: "queued", Sync: queued.ID}, nil
}

type repositoryTimer struct {
	Source     string `json:"sourceId"`
	Generation int64  `json:"generation"`
}

func repositoryTimers(location string) string { return "repository-sync-timers/" + location }
func timerID(project, source string) string   { return digest(project + "\x00" + source)[7:] }
func (s *Service) configureRepositoryTimer(ctx context.Context, source *RepositorySource, now time.Time) ([]storage.Change, error) {
	changes := []storage.Change{}
	bucket, id := repositoryTimers(s.Location), timerID(source.Project, source.ID)
	existing, err := s.Store.Get(ctx, bucket, id)
	revision := int64(0)
	if err == nil {
		revision = existing.Revision
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	source.NextSyncAt = nil
	if !source.Configuration.Enabled || source.Configuration.IntervalSeconds == 0 {
		if revision > 0 {
			changes = append(changes, storage.Change{Record: existing, Delete: true})
		}
		return changes, nil
	}
	next := now.UTC().Add(time.Duration(source.Configuration.IntervalSeconds) * time.Second)
	source.NextSyncAt = &next
	index, err := storage.Put(bucket, id, source.Project, revision, repositoryTimer{Source: source.ID, Generation: source.Generation})
	if err != nil {
		return nil, err
	}
	return append(changes, index), nil
}

func (s *Service) TickRepositorySchedules(ctx context.Context, cursor string, now time.Time) (string, error) {
	if len(s.Providers) == 0 {
		return "", nil
	}
	page, err := s.Store.List(ctx, repositoryTimers(s.Location), "", cursor, 50)
	if err != nil {
		return "", err
	}
	var problems []error
	for _, index := range page.Items {
		if ctx.Err() != nil {
			return cursor, ctx.Err()
		}
		timer, err := storage.Decode[repositoryTimer](index)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		row, err := s.Store.Get(ctx, repositoryBucket(index.Project), timer.Source)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		source, err := storage.Decode[RepositorySource](row)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if source.Generation != timer.Generation || !source.Configuration.Enabled || source.Configuration.IntervalSeconds == 0 || source.NextSyncAt == nil || now.Before(*source.NextSyncAt) {
			continue
		}
		trigger := repositoryTrigger{Kind: "periodic", Generation: source.Generation, Timer: &index, Due: *source.NextSyncAt, Now: now.UTC()}
		_, err = s.enqueueRepositorySync(ctx, source.Actor, index.Project, source.ID, "periodic/"+source.NextSyncAt.UTC().Format(time.RFC3339Nano), SyncInput{}, trigger)
		if err == nil || errors.Is(err, storage.ErrConflict) {
			continue
		}
		if ctx.Err() != nil {
			return cursor, ctx.Err()
		}
		if blocked := s.advanceBlockedRepositoryTimer(ctx, row, source, now); blocked != nil && !errors.Is(blocked, storage.ErrConflict) {
			problems = append(problems, blocked)
		}
	}
	return page.Cursor, errors.Join(problems...)
}
func (s *Service) advanceBlockedRepositoryTimer(ctx context.Context, row storage.Record, source RepositorySource, now time.Time) error {
	return s.Store.Transaction(ctx, func(store *storage.Store, _ *sql.Tx) error {
		bound := &Service{Store: store, Location: s.Location}
		changes, err := bound.configureRepositoryTimer(ctx, &source, now)
		if err != nil {
			return err
		}
		source.LastError = "Periodic synchronization is blocked by current source settings or access. Review the source before its next interval."
		source.LastState = "blocked"
		updated, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, source)
		if err != nil {
			return err
		}
		return store.Commit(ctx, append(changes, updated)...)
	})
}
