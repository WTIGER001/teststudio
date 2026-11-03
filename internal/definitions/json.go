package definitions

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

func DecodeJSON(data []byte, value any, maxBytes, maxDepth, maxNodes int) error {
	if len(data) == 0 || len(data) > maxBytes || !utf8.Valid(data) {
		return errors.New("JSON document exceeds its byte limit or has invalid UTF-8")
	}
	scan := json.NewDecoder(bytes.NewReader(data))
	scan.UseNumber()
	count := 0
	var visit func(int) error
	visit = func(depth int) error {
		count++
		if depth > maxDepth || count > maxNodes {
			return errors.New("JSON document exceeds structural limits")
		}
		token, err := scan.Token()
		if err != nil {
			return errors.New("source is not valid JSON")
		}
		delimiter, compound := token.(json.Delim)
		if !compound {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for scan.More() {
				key, err := scan.Token()
				if err != nil {
					return errors.New("JSON object key is invalid")
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return errors.New("source contains duplicate object keys")
				}
				seen[name] = true
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for scan.More() {
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("JSON document has an unexpected delimiter")
		}
		_, err = scan.Token()
		return err
	}
	if err := visit(0); err != nil {
		return err
	}
	if _, err := scan.Token(); err != io.EOF {
		return errors.New("JSON document contains trailing data")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return errors.New("document contains unknown fields or invalid values")
	}
	return nil
}
