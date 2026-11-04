package workerstate

import (
	"github.com/wtiger001/teststudio/internal/storage"
	"time"
)

type Access struct {
	Deadline     time.Time
	ProjectBytes int64
	RunBytes     int64
	Records      []storage.Record
}
