package sources

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/catalog"
	"gopkg.in/yaml.v3"
)

func Decode(format, content string) (catalog.Document, error) {
	var document catalog.Document
	if len(content) == 0 || len(content) > ContentLimit || !utf8.ValidString(content) {
		return document, errors.New("source must contain valid UTF-8 and at most 1 MiB")
	}
	if format != "json" && format != "yaml" {
		return document, errors.New("source format must be json or yaml")
	}
	if format == "json" && !json.Valid([]byte(content)) {
		return document, errors.New("source is not valid JSON")
	}
	decoder := yaml.NewDecoder(bytes.NewBufferString(content))
	var root, trailing yaml.Node
	if err := decoder.Decode(&root); err != nil || len(root.Content) != 1 {
		return document, errors.New("source is not a valid document")
	}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return document, errors.New("source must contain exactly one document")
	}
	count := 0
	value, err := jsonNode(root.Content[0], 0, &count)
	if err != nil {
		return document, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return document, errors.New("source contains unsupported values")
	}
	strict := json.NewDecoder(bytes.NewReader(data))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&document); err != nil {
		return document, errors.New("source contains unknown fields or invalid catalog values")
	}
	return document, nil
}

func jsonNode(node *yaml.Node, depth int, count *int) (any, error) {
	*count++
	if depth > 32 || *count > 100000 {
		return nil, errors.New("source document exceeds structural limits")
	}
	switch node.Kind {
	case yaml.MappingNode:
		if node.Tag != "!!map" {
			break
		}
		result := map[string]any{}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" {
				return nil, errors.New("source object keys must be strings; merge keys are not supported")
			}
			if _, exists := result[key.Value]; exists {
				return nil, errors.New("source contains duplicate object keys")
			}
			value, err := jsonNode(node.Content[i+1], depth+1, count)
			if err != nil {
				return nil, err
			}
			result[key.Value] = value
		}
		return result, nil
	case yaml.SequenceNode:
		if node.Tag != "!!seq" {
			break
		}
		result := []any{}
		for _, child := range node.Content {
			value, err := jsonNode(child, depth+1, count)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str", "!!null", "!!bool", "!!int", "!!float":
			var value any
			if err := node.Decode(&value); err != nil {
				return nil, errors.New("source scalar is invalid")
			}
			return value, nil
		}
	}
	return nil, errors.New("source aliases, custom tags, and non-JSON values are not supported")
}
