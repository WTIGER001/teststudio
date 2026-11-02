package execution

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func JSONField(value any, path string) (any, bool) {
	for _, part := range strings.Split(path, ".") {
		switch current := value.(type) {
		case map[string]any:
			next, ok := current[part]
			if !ok {
				return nil, false
			}
			value = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(current) {
				return nil, false
			}
			value = current[index]
		default:
			return nil, false
		}
	}
	return value, true
}

func assertBody(body []byte, required, forbidden string, expect map[string]any) error {
	if required != "" && !bytes.Contains(body, []byte(required)) {
		return errors.New("required response text was not present")
	}
	if forbidden != "" && bytes.Contains(body, []byte(forbidden)) {
		return errors.New("forbidden response text was present")
	}
	if len(expect) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return errors.New("response is not valid JSON")
	}
	for field, want := range expect {
		actual, ok := JSONField(value, field)
		if !ok {
			return fmt.Errorf("JSON field %s is missing", field)
		}
		a, err := json.Marshal(actual)
		if err != nil {
			return err
		}
		b, err := json.Marshal(want)
		if err != nil {
			return err
		}
		if !bytes.Equal(a, b) {
			return fmt.Errorf("JSON field %s did not match its expected value", field)
		}
	}
	return nil
}
