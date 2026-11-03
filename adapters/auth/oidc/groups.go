package oidc

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

func mappedGroups(provider Provider, claims map[string]json.RawMessage) ([]string, error) {
	if _, present := claims["hasgroups"]; present {
		return nil, errors.New("provider requires an explicit group resolver")
	}
	if _, present := claims[provider.GroupClaim+"@odata.nextLink"]; present {
		return nil, errors.New("provider returned incomplete group membership")
	}
	if data, present := claims["_claim_names"]; present {
		var distributed map[string]json.RawMessage
		if err := json.Unmarshal(data, &distributed); err != nil || distributed == nil {
			return nil, errors.New("provider returned invalid distributed claims")
		}
		if _, present := distributed[provider.GroupClaim]; present {
			return nil, errors.New("provider returned incomplete group membership")
		}
	}
	data, present := claims[provider.GroupClaim]
	if !present || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, errors.New("complete group membership is required")
	}
	var external []string
	if err := json.Unmarshal(data, &external); err != nil || len(external) > 4096 {
		return nil, errors.New("complete bounded group membership is required")
	}
	groups := []string{}
	seen := map[string]bool{}
	for _, value := range external {
		if strings.TrimSpace(value) == "" || len(value) > 512 {
			return nil, errors.New("external groups require bounded nonempty identifiers")
		}
		if group := provider.GroupMappings[value]; group != "" && !seen[group] {
			groups = append(groups, group)
			seen[group] = true
		}
	}
	return groups, nil
}
