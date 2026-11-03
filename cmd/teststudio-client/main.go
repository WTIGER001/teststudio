package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type schema struct {
	Ref                  string             `yaml:"$ref"`
	Type                 any                `yaml:"type"`
	Properties           map[string]*schema `yaml:"properties"`
	AdditionalProperties yaml.Node          `yaml:"additionalProperties"`
	Required             []string           `yaml:"required"`
	Items                *schema            `yaml:"items"`
	Enum                 []any              `yaml:"enum"`
	OneOf                []*schema          `yaml:"oneOf"`
}
type parameter struct {
	Ref      string  `yaml:"$ref"`
	Name     string  `yaml:"name"`
	In       string  `yaml:"in"`
	Required bool    `yaml:"required"`
	Schema   *schema `yaml:"schema"`
}
type content struct {
	Content map[string]struct {
		Schema *schema `yaml:"schema"`
	} `yaml:"content"`
}
type operation struct {
	ID         string             `yaml:"operationId"`
	Parameters []parameter        `yaml:"parameters"`
	Body       *content           `yaml:"requestBody"`
	Responses  map[string]content `yaml:"responses"`
}
type document struct {
	Version string `yaml:"openapi"`
	Info    struct {
		Version string `yaml:"version"`
	} `yaml:"info"`
	Paths      map[string]map[string]operation `yaml:"paths"`
	Components struct {
		Schemas    map[string]*schema   `yaml:"schemas"`
		Parameters map[string]parameter `yaml:"parameters"`
	} `yaml:"components"`
}

func quote(value string) string { data, _ := json.Marshal(value); return string(data) }
func keys[T any](values map[string]T) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	slices.Sort(result)
	return result
}

func schemaType(value *schema) (string, error) {
	if value == nil {
		return "", errors.New("a schema is required")
	}
	if value.Ref != "" {
		if !strings.HasPrefix(value.Ref, "#/components/schemas/") {
			return "", errors.New("only local schema references are supported")
		}
		return strings.TrimPrefix(value.Ref, "#/components/schemas/"), nil
	}
	if len(value.Enum) > 0 {
		types := []string{}
		for _, item := range value.Enum {
			data, err := json.Marshal(item)
			if err != nil {
				return "", err
			}
			types = append(types, string(data))
		}
		return strings.Join(types, " | "), nil
	}
	if len(value.OneOf) > 0 {
		types := []string{}
		for _, item := range value.OneOf {
			t, err := schemaType(item)
			if err != nil {
				return "", err
			}
			types = append(types, t)
		}
		return strings.Join(types, " | "), nil
	}
	if types, ok := value.Type.([]any); ok {
		result := []string{}
		for _, kind := range types {
			copy := *value
			copy.Type = kind
			t, err := schemaType(&copy)
			if err != nil {
				return "", err
			}
			result = append(result, t)
		}
		return strings.Join(result, " | "), nil
	}
	switch value.Type {
	case "string":
		return "string", nil
	case "integer", "number":
		return "number", nil
	case "boolean":
		return "boolean", nil
	case "null":
		return "null", nil
	case "array":
		t, err := schemaType(value.Items)
		return "(" + t + ")[]", err
	case "object":
		if value.AdditionalProperties.Kind == yaml.MappingNode {
			if len(value.Properties) != 0 {
				return "", errors.New("mixed fixed and dictionary properties require an explicit schema")
			}
			var item schema
			if err := value.AdditionalProperties.Decode(&item); err != nil {
				return "", err
			}
			t, err := schemaType(&item)
			return "{ [key: string]: " + t + " }", err
		}
		fields := []string{}
		for _, name := range keys(value.Properties) {
			t, err := schemaType(value.Properties[name])
			if err != nil {
				return "", err
			}
			optional := "?"
			if slices.Contains(value.Required, name) {
				optional = ""
			}
			fields = append(fields, quote(name)+optional+": "+t)
		}
		return "{ " + strings.Join(fields, "; ") + " }", nil
	default:
		return "", fmt.Errorf("unsupported schema type %v", value.Type)
	}
}

func generate(data []byte) ([]byte, error) {
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Version != "3.1.0" {
		return nil, errors.New("OpenAPI 3.1.0 is required")
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "export const accessContractVersion = %s\n\n", quote(doc.Info.Version))
	for _, name := range keys(doc.Components.Schemas) {
		t, err := schemaType(doc.Components.Schemas[name])
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", name, err)
		}
		fmt.Fprintf(&out, "export type %s = %s\n", name, t)
	}
	requests, responses, routes := []string{}, []string{}, []string{}
	seen := map[string]bool{}
	for _, path := range keys(doc.Paths) {
		for _, method := range keys(doc.Paths[path]) {
			op := doc.Paths[path][method]
			if op.ID == "" || seen[op.ID] {
				return nil, errors.New("operation identifiers must be nonempty and unique")
			}
			seen[op.ID] = true
			fields, query := []string{}, []string{}
			queryRequired := false
			for _, p := range op.Parameters {
				if p.Ref != "" {
					ref := strings.TrimPrefix(p.Ref, "#/components/parameters/")
					var ok bool
					p, ok = doc.Components.Parameters[ref]
					if !ok {
						return nil, errors.New("unresolved parameter reference")
					}
				}
				t, err := schemaType(p.Schema)
				if err != nil {
					return nil, err
				}
				optional := "?"
				if p.Required {
					optional = ""
				}
				switch p.In {
				case "path":
					fields = append(fields, quote(p.Name)+optional+": "+t)
				case "query":
					query = append(query, quote(p.Name)+optional+": "+t)
					queryRequired = queryRequired || p.Required
				case "header":
					if p.Name == "If-Match" {
						fields = append(fields, "revision: number")
					} else if p.Name != "Origin" {
						return nil, fmt.Errorf("unsupported request header %s", p.Name)
					}
				default:
					return nil, fmt.Errorf("unsupported parameter location %s", p.In)
				}
			}
			if len(query) > 0 {
				optional := "?"
				if queryRequired {
					optional = ""
				}
				fields = append(fields, "query"+optional+": { "+strings.Join(query, "; ")+" }")
			}
			if op.Body != nil {
				t, err := schemaType(op.Body.Content["application/json"].Schema)
				if err != nil {
					return nil, err
				}
				fields = append(fields, "body: "+t)
			}
			t, err := schemaType(op.Responses["200"].Content["application/json"].Schema)
			if err != nil {
				return nil, fmt.Errorf("operation %s success response: %w", op.ID, err)
			}
			requests = append(requests, "  "+op.ID+": { "+strings.Join(fields, "; ")+" }")
			responses = append(responses, "  "+op.ID+": "+t)
			routes = append(routes, "  "+op.ID+": { method: "+quote(strings.ToUpper(method))+", path: "+quote(path)+" }")
		}
	}
	fmt.Fprintf(&out, "\nexport interface AccessRequests {\n%s\n}\n\nexport interface AccessResponses {\n%s\n}\n\nexport const accessOperations = {\n%s\n} as const\n", strings.Join(requests, "\n"), strings.Join(responses, "\n"), strings.Join(routes, ",\n"))
	return out.Bytes(), nil
}

func main() {
	check := flag.Bool("check", false, "Check that the generated identity API client types match OpenAPI")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(check bool) error {
	data, err := os.ReadFile("contracts/openapi.yaml")
	if err != nil {
		return err
	}
	generated, err := generate(data)
	if err != nil {
		return err
	}
	target := "web/src/app/core/generated/access.types.ts"
	if check {
		actual, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if !bytes.Equal(actual, generated) {
			return errors.New("identity client types are stale; run go run ./cmd/teststudio-client")
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, generated, 0o644)
}
