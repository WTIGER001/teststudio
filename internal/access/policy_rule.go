package access

import (
	"errors"
	"slices"
	"time"

	permissions "github.com/wtiger001/go-permissions"
	"github.com/wtiger001/teststudio/internal/authorization"
)

func normalizePolicyRule(project string, rule RoleRule) (RoleRule, error) {
	if !slices.Contains(authorization.PermissionNames(), rule.Permission) || (rule.Effect != permissions.EffectAllow && rule.Effect != permissions.EffectDeny) {
		return rule, errors.New("each rule requires a registered permission and allow or deny effect")
	}
	if project != "" && authorization.InstallationPermission(rule.Permission) {
		return rule, errors.New("project policies cannot contain installation actions")
	}
	if rule.Object != "" && !validPolicyObject(rule.Object) {
		return rule, errors.New("rule objects require an exact supported resource identifier")
	}
	expiry, err := normalizePolicyExpiry(rule.ExpiresAt)
	rule.ExpiresAt = expiry
	return rule, err
}

func normalizePolicyExpiry(value *time.Time) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	expiry := value.UTC().Truncate(time.Microsecond)
	if expiry.Year() < 1 || expiry.Year() > 9999 {
		return nil, errors.New("expiry must use a UTC year from 1 through 9999")
	}
	return &expiry, nil
}
