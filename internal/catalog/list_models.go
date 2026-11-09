package catalog

import (
	"encoding/json"
	"time"
)

type ListQuery struct {
	Query         string `json:"query,omitempty"`
	Suite         string `json:"suite,omitempty"`
	Application   string `json:"application,omitempty"`
	Category      string `json:"category,omitempty"`
	Descendants   bool   `json:"descendants"`
	Uncategorized bool   `json:"uncategorized,omitempty"`
	Tag           string `json:"tag,omitempty"`
	Owner         string `json:"owner,omitempty"`
	Type          string `json:"type,omitempty"`
	Runner        string `json:"runner,omitempty"`
	State         string `json:"state"`
	Safety        string `json:"safety,omitempty"`
	Source        string `json:"source,omitempty"`
	Sort          string `json:"sort"`
	Limit         int    `json:"limit"`
}

type ListEntry struct {
	SourceReady bool            `json:"sourceReady"`
	ID          string          `json:"id"`
	Project     string          `json:"projectId"`
	Suite       string          `json:"suiteId"`
	SuiteName   string          `json:"suiteName"`
	Revision    int64           `json:"revision"`
	Data        json.RawMessage `json:"data"`
	Updated     time.Time       `json:"updatedAt"`
}

type CategoryNode struct {
	Path     string          `json:"path"`
	Label    string          `json:"label"`
	Count    int             `json:"count"`
	Children []*CategoryNode `json:"children"`
}

type CatalogCounts struct {
	Total   int `json:"total"`
	Enabled int `json:"enabled"`
	Safe    int `json:"safe"`
	Suites  int `json:"suites"`
}
type ListOptions struct {
	Application []string `json:"application"`
	Tag         []string `json:"tag"`
	Owner       []string `json:"owner"`
	Runner      []string `json:"runner"`
	Safety      []string `json:"safety"`
	Source      []string `json:"source"`
	Type        []string `json:"type"`
}

type ListPage struct {
	Items      []ListEntry     `json:"items"`
	Cursor     string          `json:"cursor,omitempty"`
	Total      int             `json:"total"`
	Matched    int             `json:"matched"`
	Counts     CatalogCounts   `json:"counts"`
	Categories []*CategoryNode `json:"categories"`
	Options    ListOptions     `json:"options"`
}

type listItem struct {
	Entry ListEntry
	Test  Test
}
type listCursor struct {
	Offset   int       `json:"offset"`
	Snapshot string    `json:"snapshot"`
	Expires  time.Time `json:"expiresAt"`
}

const UncategorizedCategory = "\x00uncategorized"
