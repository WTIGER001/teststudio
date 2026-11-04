package installstate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	permissions "github.com/wtiger001/go-permissions"
)

type Permission struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type Assignment struct {
	Principal permissions.PrincipalRef   `json:"principal"`
	Value     permissions.RoleAssignment `json:"value"`
}

type Edge struct {
	Parent string `json:"parent"`
	Child  string `json:"child"`
}

func IsNativeBucket(bucket string) bool {
	switch bucket {
	case "permission-roles", "permission-grants", "permission-inheritance", "permission-assignments", "permission-disabled", "permission-graph":
		return true
	}
	return false
}

func Encode(kind string, value any) (Permission, error) {
	data, err := json.Marshal(value)
	return Permission{Kind: kind, Data: data}, err
}

func Decode[T any](data json.RawMessage) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return value, errors.New("one JSON value is required")
	}
	return value, nil
}
