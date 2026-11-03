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

func TestIdentityClientRequiresTheQueryForMandatoryPermissionInputs(t *testing.T) {
	data, err := os.ReadFile("../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := generate(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"explainOwnPermission", "explainUserPermission"} {
		start := strings.Index(string(generated), "  "+operation+": {")
		if start < 0 {
			t.Fatal("missing explanation request", operation)
		}
		line := strings.SplitN(string(generated[start:]), "\n", 2)[0]
		if !strings.Contains(line, "query: {") || strings.Contains(line, "query?:") || !strings.Contains(line, `"permission": string`) || !strings.Contains(line, `"cursor"?: string`) {
			t.Fatal("required permission input became optional", line)
		}
	}
}

func TestIdentityClientPreservesStringDictionaryValues(t *testing.T) {
	var value schema
	if err := yaml.Unmarshal([]byte("type: object\nadditionalProperties: {type: string}\n"), &value); err != nil {
		t.Fatal(err)
	}
	result, err := schemaType(&value)
	if err != nil || result != "{ [key: string]: string }" {
		t.Fatal("dictionary values lost their type", result, err)
	}
}
