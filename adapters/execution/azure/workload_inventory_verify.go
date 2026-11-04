package azureworker

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func inventoryRoot(id string) ([]string, bool) {
	if len(id) > 2048 {
		return nil, false
	}
	parts := strings.Split(id, "/")
	if len(parts) != 9 || parts[0] != "" || !strings.EqualFold(parts[1], "subscriptions") || !guid.MatchString(strings.ToLower(parts[2])) || !strings.EqualFold(parts[3], "resourceGroups") || !inventoryGroup(parts[4]) || !strings.EqualFold(parts[5], "providers") || !strings.EqualFold(parts[6], "Microsoft.App") || !endpointChildName.MatchString(parts[8]) {
		return nil, false
	}
	return parts, true
}

func inventoryGroup(group string) bool {
	if group == "" || !utf8.ValidString(group) || utf8.RuneCountInString(group) > 90 || strings.HasSuffix(group, ".") {
		return false
	}
	for _, r := range group {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) && !strings.ContainsRune("_-.()", r) {
			return false
		}
	}
	return true
}

func validatePlacement(c Config, p WorkloadPlacement, kind string) error {
	resource, ok := inventoryRoot(p.ResourceID)
	if !ok {
		return hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(resource[2], c.Subscription) || !strings.EqualFold(resource[7], kind) {
		return hostedexecution.ErrOwnership
	}
	environment, ok := inventoryRoot(p.EnvironmentID)
	if !ok || !strings.EqualFold(environment[7], "managedEnvironments") && !strings.EqualFold(environment[7], "connectedEnvironments") {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func inventoryProjection(c Config, id, name, resourceType, kind, primary, legacy string) (WorkloadPlacement, error) {
	if primary != "" && legacy != "" && !strings.EqualFold(primary, legacy) {
		return WorkloadPlacement{}, hostedexecution.ErrOwnership
	}
	environment := primary
	if environment == "" {
		environment = legacy
	}
	p := WorkloadPlacement{ResourceID: id, EnvironmentID: environment}
	if err := validatePlacement(c, p, kind); err != nil {
		return WorkloadPlacement{}, err
	}
	parts, _ := inventoryRoot(id)
	if name == "" || resourceType == "" {
		return WorkloadPlacement{}, hostedexecution.ErrUnavailable
	}
	if !strings.EqualFold(name, parts[8]) || !strings.EqualFold(resourceType, "Microsoft.App/"+kind) {
		return WorkloadPlacement{}, hostedexecution.ErrOwnership
	}
	return p, nil
}

func (g *WorkloadInventoryGuard) verifyPlacements(observed []WorkloadPlacement, kind string) error {
	if len(observed) > workloadInventoryItems {
		return hostedexecution.ErrUnavailable
	}
	found := false
	seen := map[string]bool{}
	for _, placement := range observed {
		if err := validatePlacement(g.worker, placement, kind); err != nil {
			return err
		}
		id := inventoryIdentityKey(placement.ResourceID)
		if seen[id] {
			return hostedexecution.ErrOwnership
		}
		seen[id] = true
		selected := strings.EqualFold(placement.ResourceID, g.settings.Application)
		exchange := strings.EqualFold(placement.EnvironmentID, g.settings.environment())
		if kind == "containerApps" && selected {
			if !exchange {
				return hostedexecution.ErrOwnership
			}
			found = true
		} else if exchange {
			return hostedexecution.ErrOwnership
		}
	}
	if kind == "containerApps" && !found {
		return hostedexecution.ErrUnavailable
	}
	return nil
}

func inventoryIdentityKey(id string) string {
	var key strings.Builder
	key.Grow(len(id))
	for _, r := range id {
		canonical := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < canonical {
				canonical = next
			}
		}
		key.WriteRune(canonical)
	}
	return key.String()
}
