package runtimeconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

const configurationLimit = 64 << 10

func LoadJSON(filename, variable, label string, lookup func(string) (string, bool), destination any) (bool, error) {
	failure := func(reason string) (bool, error) { return false, errors.New(label + " configuration " + reason) }
	if filename != "" && variable != "" {
		return failure("requires either a file or an environment reference")
	}
	if filename == "" && variable == "" {
		return false, nil
	}
	var data []byte
	if variable != "" {
		if !environmentName.MatchString(variable) {
			return failure("must name a valid environment variable")
		}
		value, found := lookup(variable)
		if !found || strings.TrimSpace(value) == "" {
			return failure("environment variable is missing or empty")
		}
		if len(value) > configurationLimit {
			return failure("exceeds its read limit")
		}
		data = []byte(value)
	} else {
		file, err := os.Open(filename)
		if err != nil {
			return failure("cannot be read")
		}
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, configurationLimit+1))
		if err != nil || len(data) > configurationLimit {
			return failure("exceeds its read limit")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return failure("must be valid JSON with supported fields")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return failure("must contain one JSON value")
	}
	return true, nil
}
