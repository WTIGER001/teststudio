package projects

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/secrets"
)

type Requirement struct {
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	ValueType   string  `json:"valueType,omitempty"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Default     *string `json:"defaultValue,omitempty"`
}

type ConfigurationItem struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Readiness struct {
	Ready bool                `json:"ready"`
	Items []ConfigurationItem `json:"items"`
}

func DefaultSuiteID(application string) string {
	if len(application) <= 120 {
		return application + "-default"
	}
	sum := sha256.Sum256([]byte(application))
	return fmt.Sprintf("%s-%x-default", application[:100], sum[:8])
}

func ValidateRequirements(requirements []Requirement) error {
	if len(requirements) > 64 {
		return fmt.Errorf("an application supports at most 64 configuration requirements")
	}
	seen := map[string]bool{}
	for _, r := range requirements {
		if !domain.ValidID(r.Name) || seen[r.Name] || len(r.Description) > 4096 || !slices.Contains([]string{"endpoint", "parameter", "secret"}, r.Kind) {
			return fmt.Errorf("configuration requirements need unique names, a supported kind, and bounded descriptions")
		}
		seen[r.Name] = true
		if r.Kind != "parameter" && (r.Default != nil || r.ValueType != "") {
			return fmt.Errorf("only parameters accept a type or default value")
		}
		if r.Kind == "parameter" && !slices.Contains([]string{"", "text", "url", "integer", "boolean"}, r.ValueType) {
			return fmt.Errorf("unsupported parameter type")
		}
		if r.Default != nil && parameterProblem(r.ValueType, *r.Default) != "" {
			return fmt.Errorf("parameter %s has an invalid default", r.Name)
		}
	}
	return nil
}

func ValidateParameters(parameters map[string]string) error {
	if len(parameters) > 64 {
		return fmt.Errorf("an environment supports at most 64 parameters")
	}
	for name, value := range parameters {
		if !domain.ValidID(name) || len(value) > 4096 || strings.ContainsFunc(value, unicode.IsControl) {
			return fmt.Errorf("parameters need valid names and values within 4096 bytes without control characters")
		}
	}
	return nil
}

func parameterProblem(kind, value string) string {
	if strings.TrimSpace(value) == "" || len(value) > 4096 || strings.ContainsFunc(value, unicode.IsControl) {
		return "Enter a nonempty value within 4096 bytes without control characters."
	}
	switch kind {
	case "url":
		u, err := url.Parse(value)
		if err != nil || u.Hostname() == "" || u.User != nil || u.Scheme != "https" || u.Fragment != "" {
			return "Enter an HTTPS URL without embedded credentials or a fragment."
		}
	case "integer":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return "Enter a whole number."
		}
	case "boolean":
		if value != "true" && value != "false" {
			return "Choose true or false."
		}
	}
	return ""
}

func EffectiveParameters(application Application, environment Environment) map[string]string {
	values := map[string]string{}
	for _, requirement := range application.Requirements {
		if requirement.Kind == "parameter" && requirement.Default != nil {
			values[requirement.Name] = *requirement.Default
		}
	}
	for name, value := range environment.Parameters {
		values[name] = value
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func CheckConfiguration(ctx context.Context, provider secrets.Provider, project string, application Application, environment Environment, requirements []Requirement, resolve bool) Readiness {
	result := Readiness{Ready: true, Items: []ConfigurationItem{}}
	values := EffectiveParameters(application, environment)
	var refs []secrets.Reference
	if provider != nil {
		refs, _ = provider.References(ctx, project, secrets.APICredential)
	}
	seen := map[string]bool{}
	for _, r := range requirements {
		key := r.Kind + "/" + r.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		item := ConfigurationItem{Name: r.Name, Kind: r.Kind, Status: "ready"}
		declaration := slices.IndexFunc(application.Requirements, func(value Requirement) bool { return value.Name == r.Name })
		if declaration >= 0 && application.Requirements[declaration].Kind != r.Kind || r.Kind == "parameter" && declaration < 0 {
			item.Status, item.Message = "invalid", "Declare this configuration name with the correct kind in application requirements."
			result.Ready = false
			result.Items = append(result.Items, item)
			continue
		}
		switch r.Kind {
		case "parameter":
			value, found := values[r.Name]
			if !found {
				item.Status, item.Message = "missing", "Define this parameter for the environment."
			} else if problem := parameterProblem(r.ValueType, value); problem != "" {
				item.Status, item.Message = "invalid", problem
			}
		case "endpoint":
			address := environment.Endpoints[r.Name]
			u, err := url.Parse(address)
			if strings.TrimSpace(address) == "" {
				item.Status, item.Message = "missing", "Define this base URL for the environment."
			} else if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Scheme != "https" && !(u.Scheme == "http" && (environment.Stage == "local" || environment.AllowHTTP)) {
				item.Status, item.Message = "invalid", "Use a permitted HTTP or HTTPS base URL without embedded credentials."
			}
		case "secret":
			reference := environment.Credentials[r.Name]
			index := slices.IndexFunc(refs, func(ref secrets.Reference) bool { return ref.ID == reference })
			if reference == "" {
				item.Status, item.Message = "missing", "Choose an approved secret reference."
			} else if index < 0 {
				item.Status, item.Message = "unavailable", "The secret reference is unavailable."
			} else {
				approved := false
				for _, address := range environment.Endpoints {
					u, err := url.Parse(address)
					if err != nil || u == nil {
						continue
					}
					origin, originErr := browserpolicy.Origin(u.Scheme + "://" + u.Host)
					if err == nil && originErr == nil && u.Hostname() != "" && slices.Contains(refs[index].Origins, origin) {
						approved = true
					}
				}
				if !approved {
					item.Status, item.Message = "invalid", "The secret reference is not approved for an environment endpoint origin."
				} else if resolve {
					value, err := provider.Resolve(ctx, project, reference, secrets.APICredential)
					if err != nil || len(value.Bytes) == 0 {
						item.Status, item.Message = "unavailable", "The secret reference could not be resolved."
					}
					clear(value.Bytes)
				} else {
					item.Status, item.Message = "configured", "Reference configured. Availability is checked before execution."
				}
			}
		}
		if item.Status != "ready" && item.Status != "configured" {
			result.Ready = false
		}
		result.Items = append(result.Items, item)
	}
	slices.SortFunc(result.Items, func(a, b ConfigurationItem) int { return strings.Compare(a.Kind+a.Name, b.Kind+b.Name) })
	return result
}

func (r Readiness) Error(environment string) error {
	if r.Ready {
		return nil
	}
	names := []string{}
	for _, item := range r.Items {
		if item.Status != "ready" && item.Status != "configured" {
			names = append(names, item.Name+" ("+item.Status+")")
		}
	}
	return fmt.Errorf("environment %s needs configuration: %s", environment, strings.Join(names, ", "))
}
