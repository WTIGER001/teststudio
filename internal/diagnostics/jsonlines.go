package diagnostics

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func (r *Redactor) learnValue(value any, depth int) {
	if depth > 64 {
		r.overflow = true
		return
	}
	switch value := value.(type) {
	case map[string]any:
		for name, item := range value {
			if r.fields[key(name)] {
				r.leaves(item, depth+1)
				r.headerValues(key(name), item)
			} else {
				r.learnValue(item, depth+1)
			}
		}
	case []any:
		for _, item := range value {
			r.learnValue(item, depth+1)
		}
	case string:
		r.Learn(value)
	}
}

func (r *Redactor) LearnJSON(data []byte) error {
	if len(data) > MaxBytes {
		return errors.New("API structured diagnostic exceeds its redaction limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return errors.New("API structured diagnostic is invalid")
	}
	var extra any
	if !errors.Is(decoder.Decode(&extra), io.EOF) {
		return errors.New("API structured diagnostic has trailing data")
	}
	r.learnValue(value, 0)
	return nil
}

func (r *Redactor) JSONLines(data []byte) ([]byte, error) {
	if len(data) > MaxBytes {
		return nil, errors.New("API JSON event stream exceeds its redaction limit")
	}
	lines := bytes.Split(bytes.TrimSpace(data), []byte{'\n'})
	if len(lines) > 4096 {
		return nil, errors.New("API JSON event stream exceeds its event limit")
	}
	for _, line := range lines {
		if err := r.LearnJSON(line); err != nil {
			return nil, err
		}
	}
	var output bytes.Buffer
	for _, line := range lines {
		clean, err := r.JSON(line)
		if err != nil {
			return nil, err
		}
		output.Write(clean)
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}
