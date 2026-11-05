package kubernetes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func normalizedNetworkSpec(kind string, input map[string]any, seconds int, stopping bool) (map[string]any, error) {
	if kind == "Pod" {
		return normalizedSpec(input, seconds, stopping)
	}
	data, err := json.Marshal(input)
	if err != nil {
		return nil, hostedexecution.ErrOwnership
	}
	var spec map[string]any
	if json.Unmarshal(data, &spec) != nil || spec == nil {
		return nil, hostedexecution.ErrOwnership
	}
	switch kind {
	case "NetworkPolicy":
		for _, name := range []string{"ingress", "egress"} {
			if spec[name] == nil {
				spec[name] = []any{}
			}
		}
	case "Service":
		for name, value := range map[string]any{"sessionAffinity": "None", "internalTrafficPolicy": "Cluster", "ipFamilyPolicy": "SingleStack", "publishNotReadyAddresses": false} {
			if spec[name] == value {
				delete(spec, name)
			}
		}
		if values, ok := spec["clusterIPs"].([]any); ok && len(values) == 1 && values[0] == "None" {
			delete(spec, "clusterIPs")
		}
		if values, ok := spec["ipFamilies"].([]any); ok && len(values) == 1 && (values[0] == "IPv4" || values[0] == "IPv6") {
			delete(spec, "ipFamilies")
		}
	default:
		return nil, hostedexecution.ErrOwnership
	}
	return spec, nil
}

func networkListObject(value networkObject, kind string) (networkObject, error) {
	api := "v1"
	if kind == "NetworkPolicy" {
		api = "networking.k8s.io/v1"
	}
	if value.Kind != "" && value.Kind != kind || value.APIVersion != "" && value.APIVersion != api {
		return value, hostedexecution.ErrOwnership
	}
	value.Kind, value.APIVersion = kind, api
	return value, nil
}

func resourceIndex(record dispatch, kind, name string) int {
	if record.Network == nil {
		return -1
	}
	for i, value := range record.Network.Resources {
		if value.Kind == kind && value.Name == name {
			return i
		}
	}
	return -1
}

func (p *Provider) networkPath(kind, name string) string {
	base := "/api/v1/namespaces/" + p.config.Namespace
	switch kind {
	case "Pod":
		base += "/pods"
	case "Service":
		base += "/services"
	case "NetworkPolicy":
		base = "/apis/networking.k8s.io/v1/namespaces/" + p.config.Namespace + "/networkpolicies"
	default:
		return ""
	}
	if name != "" {
		base += "/" + name
	}
	return base
}

func (p *Provider) verifyNetworkObject(value networkObject, record dispatch) error {
	index := resourceIndex(record, value.Kind, value.Metadata.Name)
	if index < 0 {
		return hostedexecution.ErrOwnership
	}
	proof := record.Network.Resources[index]
	if !proof.Attempted {
		return hostedexecution.ErrOwnership
	}
	api := "v1"
	if value.Kind == "NetworkPolicy" {
		api = "networking.k8s.io/v1"
	}
	var ref execution.Reference
	if value.APIVersion != api || value.Metadata.Namespace != p.config.Namespace || value.Metadata.UID == "" || value.Metadata.Version == "" || len(value.Metadata.OwnerReferences) != 0 || len(value.Metadata.Annotations[ownerAnnotation]) > 4096 || json.Unmarshal([]byte(value.Metadata.Annotations[ownerAnnotation]), &ref) != nil || ref != record.Reference || proof.UID != "" && proof.UID != value.Metadata.UID || value.Metadata.Annotations[specAnnotation] != proof.SpecHash {
		return hostedexecution.ErrOwnership
	}
	for key, want := range networkLabels(ref, value.Metadata.Name == ref.Gateway) {
		if value.Metadata.Labels[key] != want {
			return hostedexecution.ErrOwnership
		}
	}
	deadline, err := time.Parse(time.RFC3339Nano, value.Metadata.Annotations[deadlineAnnotation])
	if err != nil || !deadline.Equal(record.Deadline) {
		return hostedexecution.ErrOwnership
	}
	if value.Kind == "Pod" && value.Metadata.Annotations[secondsAnnotation] != strconv.Itoa(record.ActiveSeconds) {
		return hostedexecution.ErrOwnership
	}
	spec, err := normalizedNetworkSpec(value.Kind, value.Spec, record.ActiveSeconds, record.Stopping)
	if err != nil || fingerprint(spec) != proof.SpecHash {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *Provider) rememberNetworkObject(ctx context.Context, ref execution.Reference, value networkObject) (dispatch, error) {
	return p.update(ctx, ref, func(record *dispatch) error {
		if err := p.verifyNetworkObject(value, *record); err != nil {
			return err
		}
		index := resourceIndex(*record, value.Kind, value.Metadata.Name)
		record.Network.Resources[index].UID = value.Metadata.UID
		return nil
	})
}

func (p *Provider) checkPolicies(ctx context.Context) error {
	var list struct {
		Items    []networkObject `json:"items"`
		Metadata struct {
			Continue string `json:"continue"`
		} `json:"metadata"`
	}
	if err := p.request(ctx, http.MethodGet, p.networkPath("NetworkPolicy", ""), url.Values{"limit": {"1000"}}, nil, &list); err != nil || list.Metadata.Continue != "" || len(list.Items) < 1 {
		return hostedexecution.ErrConfiguration
	}
	base := false
	for _, value := range list.Items {
		value, err := networkListObject(value, "NetworkPolicy")
		if err != nil {
			return err
		}
		if value.Metadata.Name == p.config.PolicyName {
			if base {
				return hostedexecution.ErrConfiguration
			}
			base = true
			if value.Spec["ingress"] == nil {
				value.Spec["ingress"] = []any{}
			}
			if fingerprint(value.Spec) != fingerprint(p.policy()) {
				return hostedexecution.ErrConfiguration
			}
			continue
		}
		var ref execution.Reference
		if p.config.Gateway == nil || json.Unmarshal([]byte(value.Metadata.Annotations[ownerAnnotation]), &ref) != nil || ref.Gateway == "" {
			return hostedexecution.ErrConfiguration
		}
		record, _, err := p.load(ctx, ref)
		if err != nil || p.verifyNetworkObject(value, record) != nil {
			return hostedexecution.ErrConfiguration
		}
	}
	if !base {
		return hostedexecution.ErrConfiguration
	}
	return nil
}
