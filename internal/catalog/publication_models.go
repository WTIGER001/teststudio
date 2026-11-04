package catalog

import (
	"github.com/wtiger001/teststudio/internal/timing"
	"time"
)

type PublicationChange struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Action        string   `json:"action"`
	Fields        []string `json:"fields"`
	Before        *Test    `json:"before,omitempty"`
	After         *Test    `json:"after,omitempty"`
	SafetyChanged bool     `json:"safetyChanged"`
}

type PublicationSchedule struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	Release       string `json:"releaseId,omitempty"`
	FollowsActive bool   `json:"followsActive"`
}

type PublicationSettings struct {
	Artifact Artifact      `json:"artifact"`
	Timing   timing.Policy `json:"timing"`
	Capture  CapturePolicy `json:"capture"`
}

type PublicationReview struct {
	BeforeSettings  *PublicationSettings  `json:"beforeSettings,omitempty"`
	AfterSettings   PublicationSettings   `json:"afterSettings"`
	ID              string                `json:"id"`
	Project         string                `json:"projectId"`
	Suite           string                `json:"suiteId"`
	Draft           string                `json:"draftId"`
	DraftRevision   int64                 `json:"draftRevision"`
	Active          string                `json:"activeReleaseId,omitempty"`
	Changes         []PublicationChange   `json:"changes"`
	Unchanged       int                   `json:"unchanged"`
	Total           int                   `json:"total"`
	Enabled         int                   `json:"enabled"`
	ArtifactChanged bool                  `json:"artifactChanged"`
	TimingChanged   bool                  `json:"timingChanged"`
	CaptureChanged  bool                  `json:"captureChanged"`
	Issues          []Issue               `json:"issues"`
	Schedules       []PublicationSchedule `json:"schedules"`
	CreatedAt       time.Time             `json:"createdAt"`
	ExpiresAt       time.Time             `json:"expiresAt"`
}

type publicationReviewRecord struct {
	Review          PublicationReview `json:"review"`
	Actor           string            `json:"actor"`
	SuiteRevision   int64             `json:"suiteRevision"`
	ProjectRevision int64             `json:"projectRevision"`
	Artifact        Artifact          `json:"artifact"`
	HeadRevision    int64             `json:"headRevision"`
	ScheduleDigest  string            `json:"scheduleDigest"`
	TestIDs         []string          `json:"testIds"`
	Release         string            `json:"releaseId,omitempty"`
	Key             string            `json:"key,omitempty"`
	Activate        bool              `json:"activate"`
}
