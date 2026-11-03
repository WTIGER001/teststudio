package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestIdentityClientStaysSynchronizedWithOpenAPI(t *testing.T) {
	data, err := os.ReadFile("../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := generate(data)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile("../../web/src/app/core/generated/access.types.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, generated) {
		t.Fatal("identity API types are stale; run go run ./cmd/teststudio-client")
	}
}

func TestIdentityClientPreservesNullableExpiryAndRequiredFields(t *testing.T) {
	var value schema
	if err := yaml.Unmarshal([]byte("type: object\nrequired: [id]\nproperties:\n  id: {type: string}\n  expiry: {type: [string, 'null']}\n"), &value); err != nil {
		t.Fatal(err)
	}
	result, err := schemaType(&value)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, `"id": string`) || !strings.Contains(result, `"expiry"?: string | null`) {
		t.Fatal(result)
	}
	if _, err := schemaType(&schema{Type: "unsupported"}); err == nil {
		t.Fatal("unsupported schema silently changed the contract")
	}
}
