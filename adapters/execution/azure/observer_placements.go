package azureworker

import (
	"strings"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func verifyObserverPlacements(c Config, s ObserverHostSettings, observed []WorkloadPlacement, kind string) error {
	return (identityHostBinding{s.Application, c.ControllerEnvironment, s.Identity.Identity, s.Identity.ClientID, s.Identity.PrincipalID}).verify(c, observed, kind)
}

type identityHostBinding struct{ application, environment, resource, client, principal string }

func (s identityHostBinding) verify(c Config, observed []WorkloadPlacement, kind string) error {
	if observed == nil || len(observed) > workloadInventoryItems || kind != "containerApps" && kind != "jobs" {
		return hostedexecution.ErrUnavailable
	}
	found, seen := false, map[string]bool{}
	for _, placement := range observed {
		if err := validatePlacement(c, placement, kind); err != nil {
			return err
		}
		key := inventoryIdentityKey(placement.ResourceID)
		if seen[key] {
			return hostedexecution.ErrOwnership
		}
		seen[key] = true
		if !placement.IdentityKnown || placement.Identities == nil || len(placement.Identities) > workloadIdentityItems {
			return hostedexecution.ErrUnavailable
		}
		if strings.EqualFold(placement.SystemPrincipalID, s.principal) {
			return hostedexecution.ErrOwnership
		}
		host := kind == "containerApps" && strings.EqualFold(placement.ResourceID, s.application)
		if host && !strings.EqualFold(placement.EnvironmentID, s.environment) {
			return hostedexecution.ErrOwnership
		}
		count := 0
		for _, identity := range placement.Identities {
			if !workloadIdentityReference(identity.ResourceID) {
				return hostedexecution.ErrUnavailable
			}
			selected := strings.EqualFold(identity.ResourceID, s.resource)
			alias := strings.EqualFold(identity.ClientID, s.client) || strings.EqualFold(identity.PrincipalID, s.principal)
			if !selected && !alias {
				continue
			}
			if !host || !selected {
				return hostedexecution.ErrOwnership
			}
			if identity.ClientID == "" || identity.PrincipalID == "" {
				return hostedexecution.ErrUnavailable
			}
			if !strings.EqualFold(identity.ClientID, s.client) || !strings.EqualFold(identity.PrincipalID, s.principal) {
				return hostedexecution.ErrOwnership
			}
			count++
		}
		if host {
			if count == 0 {
				return hostedexecution.ErrUnavailable
			}
			if count != 1 {
				return hostedexecution.ErrOwnership
			}
			found = true
		}
	}
	if kind == "containerApps" && !found {
		return hostedexecution.ErrUnavailable
	}
	return nil
}
