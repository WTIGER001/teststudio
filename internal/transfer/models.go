package transfer

import (
	"encoding/json"
	"errors"
	"time"
)

const Schema = "teststudio.installation/v1"
const PermissionVersion = "go-permissions/v0.0.31"
const MaxEntryBytes = 64 << 20

var ErrInvalidArchive = errors.New("installation archive is invalid or incompatible")
var ErrBusy = errors.New("finish active runs, recordings, preparation, and worker recovery before exporting")

var ErrNotEmpty = errors.New("destination must contain no installation data")

type Header struct {
	Schema            string    `json:"schemaVersion"`
	PermissionVersion string    `json:"permissionVersion"`
	MetadataVersion   int       `json:"metadataVersion"`
	Source            string    `json:"sourceDialect"`
	CreatedAt         time.Time `json:"createdAt"`
	GrantSequence     int64     `json:"grantSequence"`
}

type Stats struct {
	Records     int64            `json:"records"`
	Counters    int64            `json:"counters"`
	Permissions map[string]int64 `json:"permissions"`
	Reliability int64            `json:"reliability,omitempty"`
}

type Entry struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type Record struct {
	Bucket    string `json:"bucket"`
	ID        string `json:"id"`
	Project   string `json:"project"`
	Revision  int64  `json:"revision"`
	Payload   string `json:"payload"`
	UpdatedAt string `json:"updatedAt"`
}

type Counter struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type footer struct {
	Stats    Stats  `json:"counts"`
	Checksum string `json:"sha256"`
}
