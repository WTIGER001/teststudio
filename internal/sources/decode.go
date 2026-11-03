package sources

import (
	"github.com/wtiger001/teststudio/internal/catalog"
	"github.com/wtiger001/teststudio/internal/definitions"
)

func Decode(format, content string) (catalog.Document, error) {
	var document catalog.Document
	err := definitions.Decode(format, content, &document)
	return document, err
}
