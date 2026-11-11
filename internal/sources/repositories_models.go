package sources

import (
	"time"

	"github.com/wtiger001/teststudio/internal/sourceprovider"
)

type RepositoryBinding struct {
	Discover   bool   `json:"discover,omitempty"`
	ModuleRoot string `json:"moduleRoot,omitempty"`
	Config     string `json:"config,omitempty"`
	Project    string `json:"project,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Suite      string `json:"suiteId"`
	Manifest   string `json:"manifestPath"`
	Runner     string `json:"runner"`
	Profile    string `json:"buildProfile"`
}

type RepositoryConfig struct {
	ConfirmModeChange bool                `json:"confirmModeChange,omitempty"`
	Mode              string              `json:"accessMode,omitempty"`
	Push              bool                `json:"pushEnabled,omitempty"`
	WebhookCredential string              `json:"webhookCredentialReference,omitempty"`
	IntervalSeconds   int                 `json:"syncIntervalSeconds,omitempty"`
	Name              string              `json:"name"`
	Provider          string              `json:"provider"`
	Repository        string              `json:"repository"`
	Reference         string              `json:"reference"`
	Credential        string              `json:"credentialReference,omitempty"`
	Enabled           bool                `json:"enabled"`
	AutoActivate      bool                `json:"autoActivate"`
	Bindings          []RepositoryBinding `json:"bindings"`
}

type RepositorySource struct {
	NextSyncAt    *time.Time       `json:"nextSyncAt,omitempty"`
	ID            string           `json:"id"`
	Project       string           `json:"projectId"`
	Configuration RepositoryConfig `json:"configuration"`
	Actor         string           `json:"actor"`
	Generation    int64            `json:"generation"`
	Sequence      int64            `json:"sequence"`
	RepositoryID  string           `json:"repositoryId,omitempty"`
	LastSync      string           `json:"lastSyncId,omitempty"`
	LastCommit    string           `json:"lastCommit,omitempty"`
	LastError     string           `json:"lastError,omitempty"`
	LastState     string           `json:"lastState,omitempty"`
	LastSyncAt    *time.Time       `json:"lastSyncAt,omitempty"`
}

type SyncInput struct {
	Force bool `json:"force"`
}

type RepositorySync struct {
	Trigger          string                   `json:"trigger,omitempty"`
	Delivery         string                   `json:"deliveryId,omitempty"`
	EventDigest      string                   `json:"eventDigest,omitempty"`
	ID               string                   `json:"id"`
	Project          string                   `json:"projectId"`
	Source           string                   `json:"sourceId"`
	Actor            string                   `json:"actor"`
	RequestedBy      string                   `json:"requestedBy"`
	SourceGeneration int64                    `json:"sourceGeneration"`
	Sequence         int64                    `json:"sequence"`
	Configuration    RepositoryConfig         `json:"configuration"`
	RepositoryID     string                   `json:"repositoryId,omitempty"`
	Force            bool                     `json:"force"`
	State            string                   `json:"state"`
	Error            string                   `json:"error,omitempty"`
	Revision         *sourceprovider.Revision `json:"revision,omitempty"`
	Snapshot         string                   `json:"snapshotId,omitempty"`
	Preparations     map[string]string        `json:"preparations"`
	Worker           string                   `json:"worker,omitempty"`
	LeaseUntil       *time.Time               `json:"leaseUntil,omitempty"`
	CreatedAt        time.Time                `json:"createdAt"`
	FinishedAt       *time.Time               `json:"finishedAt,omitempty"`
}

type RepositorySnapshot struct {
	ID         string                  `json:"id"`
	Project    string                  `json:"projectId"`
	Source     string                  `json:"sourceId"`
	Origin     sourceprovider.Revision `json:"origin"`
	Content    []byte                  `json:"-"`
	Root       string                  `json:"archiveRoot"`
	Digest     string                  `json:"digest"`
	References int                     `json:"references"`
	Published  bool                    `json:"published"`
	ExpiresAt  time.Time               `json:"expiresAt"`
}

type syncIndex struct {
	Source string `json:"sourceId"`
	Sync   string `json:"syncId"`
}

func repositoryBucket(project string) string   { return "repository-sources/" + project }
func syncBucket(project, source string) string { return "repository-syncs/" + project + "/" + source }
func syncQueue(location string) string         { return "repository-sync-queue/" + location }
func syncLeases(location string) string        { return "repository-sync-leases/" + location }
func snapshotBucket(project string) string     { return "repository-snapshots/" + project }
