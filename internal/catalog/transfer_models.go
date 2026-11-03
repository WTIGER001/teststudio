package catalog

import "time"

const SchemaVersion = "teststudio.catalog/v1"
const MutationLimit = 30

type Document struct {
	Schema     string     `json:"schemaVersion"`
	Project    string     `json:"originProjectId,omitempty"`
	Suite      string     `json:"originSuiteId,omitempty"`
	ExportedAt *time.Time `json:"exportedAt,omitempty"`
	Tests      []Test     `json:"tests"`
}

type PlannedTest struct {
	Test     Test   `json:"test"`
	Before   *Test  `json:"before,omitempty"`
	Revision int64  `json:"expectedRevision"`
	Action   string `json:"action"`
}

type ImportPlan struct {
	Artifact       *Artifact     `json:"artifact,omitempty"`
	SourceSnapshot string        `json:"sourceSnapshotId,omitempty"`
	ID             string        `json:"id"`
	Project        string        `json:"projectId"`
	Suite          string        `json:"suiteId"`
	Actor          string        `json:"actor"`
	Kind           string        `json:"kind"`
	HeadRevision   int64         `json:"catalogRevision"`
	SuiteRevision  int64         `json:"suiteRevision"`
	Changes        []PlannedTest `json:"changes"`
	Issues         []Issue       `json:"issues"`
	CreatedAt      time.Time     `json:"createdAt"`
	ExpiresAt      time.Time     `json:"expiresAt"`
	AppliedAt      *time.Time    `json:"appliedAt,omitempty"`
}

type BulkRequest struct {
	Action string `json:"action"`
	Tests  []struct {
		ID       string `json:"id"`
		Revision int64  `json:"revision"`
	} `json:"tests"`
}
