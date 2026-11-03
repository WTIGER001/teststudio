package catalog

import "time"

type SavedSelectionFilter struct {
	Query       string   `json:"query,omitempty"`
	Category    string   `json:"category,omitempty"`
	Descendants bool     `json:"includeCategoryDescendants,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Application string   `json:"applicationId,omitempty"`
	Type        string   `json:"type,omitempty"`
	Runner      string   `json:"runner,omitempty"`
	Safety      string   `json:"safety,omitempty"`
	Source      string   `json:"source,omitempty"`
}

type SavedSelectionSpec struct {
	Version int                   `json:"schemaVersion"`
	Kind    string                `json:"kind"`
	IDs     []string              `json:"testIds,omitempty"`
	Filter  *SavedSelectionFilter `json:"filter,omitempty"`
}

type SavedSelection struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Spec        SavedSelectionSpec `json:"spec"`
	Archived    bool               `json:"archived"`
}

type SavedSelectionSnapshot struct {
	Definition SavedSelection `json:"definition"`
	Revision   int64          `json:"revision"`
	PreviewID  string         `json:"previewId"`
}

type SavedSelectionPreview struct {
	ID       string                 `json:"id"`
	Project  string                 `json:"projectId"`
	Suite    string                 `json:"suiteId"`
	Actor    string                 `json:"actor"`
	Release  string                 `json:"releaseId"`
	Snapshot SavedSelectionSnapshot `json:"selection"`
	Roots    []string               `json:"requestedTestIds"`
	Resolved []string               `json:"resolvedTestIds"`
	Created  time.Time              `json:"createdAt"`
	Expires  time.Time              `json:"expiresAt"`
}
