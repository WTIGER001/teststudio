package sourceprovider

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/sourcebundle"
)

type WriteRequest struct {
	Base    Revision          `json:"base"`
	Key     string            `json:"key"`
	Message string            `json:"message"`
	Files   map[string]string `json:"files"`
}
type WriteResult struct {
	Commit    string `json:"commit"`
	Branch    string `json:"branch"`
	ReviewURL string `json:"reviewUrl"`
}
type Writer interface {
	WriteMetadata(context.Context, Access, WriteRequest) (WriteResult, error)
}

func ValidateWrite(r WriteRequest) error {
	if !domain.ValidID(r.Key) || !ValidCommit(r.Base.Commit) || !strings.HasPrefix(r.Base.Reference, "heads/") || !ValidReference(r.Base.Reference) || len(strings.TrimSpace(r.Message)) < 1 || len(r.Message) > 256 || strings.ContainsAny(r.Message, "\r\n\x00") || len(r.Files) < 1 || len(r.Files) > 40 {
		return errors.New("metadata commit requires a branch, exact base, operation identifier, message, and bounded files")
	}
	total := 0
	for name, value := range r.Files {
		total += len(value)
		if !sourcebundle.SafePath(name) || !slices.Contains([]string{".json", ".yaml", ".yml"}, path.Ext(name)) || len(value) > 1<<20 {
			return errors.New("metadata writes require bounded relative JSON files")
		}
	}
	if total > 2<<20 {
		return errors.New("metadata commit exceeds 2 MiB")
	}
	return nil
}
