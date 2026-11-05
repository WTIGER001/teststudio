package azureworker

import (
	"strings"

	a "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func readDenyPattern(pattern string) bool {
	if len(pattern) == 0 || len(pattern) > 512 {
		return false
	}
	for _, c := range pattern {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("./_-*", c) {
			continue
		}
		return false
	}
	return true
}

func readDenyMatches(pattern, action string) bool {
	parts := strings.Split(strings.ToLower(pattern), "*")
	action = strings.ToLower(action)
	if len(parts) == 1 {
		return action == parts[0]
	}
	if !strings.HasPrefix(action, parts[0]) {
		return false
	}
	action = action[len(parts[0]):]
	for _, part := range parts[1 : len(parts)-1] {
		index := strings.Index(action, part)
		if index < 0 {
			return false
		}
		action = action[index+len(part):]
	}
	return strings.HasSuffix(action, parts[len(parts)-1])
}

func readDenyIntersectsRead(pattern string) bool {
	pattern = strings.ToLower(pattern)
	index := strings.LastIndex(pattern, "*")
	if index < 0 {
		return strings.HasSuffix(pattern, "/read")
	}
	tail := pattern[index+1:]
	return strings.HasSuffix("/read", tail) || strings.HasSuffix(tail, "/read")
}

func readDenyPermission(p *a.DenyAssignmentPermission) (bool, error) {
	if p == nil || len(p.Actions)+len(p.DataActions) == 0 || len(p.Actions) > 256 || len(p.NotActions) > 256 {
		return false, hostedexecution.ErrUnavailable
	}
	universalRead, all := false, false
	for _, pattern := range p.NotActions {
		if pattern == nil || !readDenyPattern(value(pattern)) {
			return false, hostedexecution.ErrUnavailable
		}
		universalRead = universalRead || strings.EqualFold(value(pattern), "*/read")
		all = all || value(pattern) == "*"
	}
	blocked := false
	for _, pattern := range p.Actions {
		if pattern == nil || !readDenyPattern(value(pattern)) {
			return false, hostedexecution.ErrUnavailable
		}
		action := value(pattern)
		read := readDenyIntersectsRead(action) && !universalRead && !all
		association := readDenyMatches(action, networkReadAssociationAction) && !all
		for _, exception := range p.NotActions {
			if strings.EqualFold(action, value(exception)) || !strings.Contains(action, "*") && readDenyMatches(value(exception), action) {
				read = false
			}
			if readDenyMatches(value(exception), networkReadAssociationAction) {
				association = false
			}
		}
		blocked = blocked || read || association
	}
	if blocked && (value(p.Condition) != "" || value(p.ConditionVersion) != "") {
		return false, hostedexecution.ErrUnavailable
	}
	return blocked, nil
}
