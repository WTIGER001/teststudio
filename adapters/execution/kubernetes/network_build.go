package kubernetes

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

func networkLabels(ref execution.Reference, gateway bool) map[string]string {
	result := labels(ref)
	if gateway {
		result[workerLabel] = "gateway-v1"
	}
	return result
}

func (p *Provider) gatewaySettings(ref execution.Reference) workerexchange.GatewaySettings {
	return workerexchange.GatewaySettings{Origin: "https://" + ref.Gateway + "." + p.config.Namespace + ".svc:8044", CertificateAuthority: p.gatewayCA}
}

func (p *Provider) gatewayPod(job hostedexecution.Job, ticket workerexchange.GatewayTicket, seconds int) networkObject {
	value := p.build(job, seconds)
	value.Metadata.Name = job.Reference.Gateway
	value.Metadata.Labels = networkLabels(job.Reference, true)
	value.Spec["serviceAccountName"] = p.config.Gateway.ServiceAccount
	volumes := []any{map[string]any{"name": "tls", "secret": map[string]any{"secretName": p.config.Gateway.TLSSecret, "defaultMode": 0440, "items": []any{map[string]any{"key": "tls.crt", "path": "tls.crt"}, map[string]any{"key": "tls.key", "path": "tls.key"}}}}}
	mounts := []any{map[string]any{"name": "tls", "mountPath": "/tls", "readOnly": true}}
	env := []any{
		map[string]any{"name": "TESTSTUDIO_GATEWAY_EXECUTION", "value": job.Reference.ID},
		map[string]any{"name": "TESTSTUDIO_GATEWAY_TOKEN", "value": ticket.Token},
		map[string]any{"name": "TESTSTUDIO_GATEWAY_INSTANCE", "valueFrom": map[string]any{"fieldRef": map[string]any{"apiVersion": "v1", "fieldPath": "metadata.uid"}}},
	}
	if name := p.config.Gateway.ExchangeTrustSecret; name != "" {
		volumes = append(volumes, map[string]any{"name": "exchange-trust", "secret": map[string]any{"secretName": name, "defaultMode": 0440, "items": []any{map[string]any{"key": "ca.crt", "path": "ca.crt"}}}})
		mounts = append(mounts, map[string]any{"name": "exchange-trust", "mountPath": "/exchange-trust", "readOnly": true})
		env = append(env, map[string]any{"name": "SSL_CERT_FILE", "value": "/exchange-trust/ca.crt"})
	}
	value.Spec["volumes"] = volumes
	value.Spec["containers"] = []any{map[string]any{
		"name": "gateway", "image": p.config.Gateway.Image, "imagePullPolicy": "IfNotPresent", "command": []string{"/gateway"}, "args": []string{"--remote-exchange=" + job.ExchangeOrigin}, "env": env,
		"securityContext": map[string]any{"allowPrivilegeEscalation": false, "readOnlyRootFilesystem": true, "capabilities": map[string]any{"drop": []string{"ALL"}}},
		"resources":       map[string]any{"requests": map[string]string{"cpu": "100m", "memory": "128Mi", "ephemeral-storage": "64Mi"}, "limits": map[string]string{"cpu": "1", "memory": "256Mi", "ephemeral-storage": "64Mi"}},
		"ports":           []any{map[string]any{"name": "proxy", "containerPort": 8044, "protocol": "TCP"}},
		"readinessProbe":  map[string]any{"exec": map[string]any{"command": []string{"/gateway", "--ready-remote"}}, "initialDelaySeconds": 1, "periodSeconds": 1, "timeoutSeconds": 1, "failureThreshold": 3, "successThreshold": 1},
		"volumeMounts":    mounts, "terminationMessagePath": "/dev/termination-log", "terminationMessagePolicy": "File",
	}}
	return networkObject{APIVersion: value.APIVersion, Kind: value.Kind, Metadata: value.Metadata, Spec: value.Spec}
}

func (p *Provider) networkObjects(job hostedexecution.Job, ticket workerexchange.GatewayTicket, seconds int) ([]networkObject, error) {
	ref, _ := json.Marshal(job.Reference)
	metadataFor := func(name string, gateway bool) metadata {
		return metadata{Name: name, Namespace: p.config.Namespace, Labels: networkLabels(job.Reference, gateway), Annotations: map[string]string{ownerAnnotation: string(ref), deadlineAnnotation: job.Ticket.Deadline.UTC().Format(time.RFC3339Nano)}}
	}
	workerSelector := map[string]any{"matchLabels": networkLabels(job.Reference, false)}
	gatewaySelector := map[string]any{"matchLabels": networkLabels(job.Reference, true)}
	objects := []networkObject{
		{APIVersion: "networking.k8s.io/v1", Kind: "NetworkPolicy", Metadata: metadataFor(job.Reference.Container, false), Spec: map[string]any{
			"podSelector": workerSelector, "policyTypes": []string{"Ingress", "Egress"}, "ingress": []any{},
			"egress": []any{map[string]any{"to": []any{map[string]any{"podSelector": gatewaySelector}}, "ports": []any{map[string]any{"protocol": "TCP", "port": 8044}}}},
		}},
		{APIVersion: "networking.k8s.io/v1", Kind: "NetworkPolicy", Metadata: metadataFor(job.Reference.Gateway, true), Spec: map[string]any{
			"podSelector": gatewaySelector, "policyTypes": []string{"Ingress", "Egress"}, "egress": []any{map[string]any{}},
			"ingress": []any{map[string]any{"from": []any{map[string]any{"podSelector": workerSelector}}, "ports": []any{map[string]any{"protocol": "TCP", "port": 8044}}}},
		}},
		{APIVersion: "v1", Kind: "Service", Metadata: metadataFor(job.Reference.Gateway, true), Spec: map[string]any{
			"type": "ClusterIP", "clusterIP": "None", "selector": networkLabels(job.Reference, true),
			"ports": []any{map[string]any{"name": "proxy", "protocol": "TCP", "port": 8044, "targetPort": "proxy"}},
		}},
		p.gatewayPod(job, ticket, seconds),
	}
	for i := range objects {
		spec, err := normalizedNetworkSpec(objects[i].Kind, objects[i].Spec, seconds, false)
		if err != nil {
			return nil, err
		}
		objects[i].Metadata.Annotations[specAnnotation] = fingerprint(spec)
		if objects[i].Kind == "Pod" {
			objects[i].Metadata.Annotations[secondsAnnotation] = strconv.Itoa(seconds)
		}
	}
	return objects, nil
}
