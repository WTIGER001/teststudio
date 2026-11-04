package kubernetes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func fingerprint(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func short(value string) string { return fingerprint(value)[:40] }

func (p *Provider) scope() string {
	return fingerprint([]string{p.config.APIOrigin, p.config.Namespace})
}

func validReference(ref execution.Reference) bool {
	if strings.HasSuffix(ref.Profile, "-discovery-v1") && (!strings.HasPrefix(ref.Installation, "preparation-") || ref.Test != "catalog" || ref.Attempt != 1) {
		return false
	}
	return domain.ValidID(ref.ID) && domain.ValidID(ref.Installation) && domain.ValidID(ref.Project) && domain.ValidID(ref.Run) && domain.ValidID(ref.Test) && domain.ValidID(ref.Worker) && ref.Attempt >= 1 && ref.Attempt <= 3 && ref.Volume == "" && ref.Gateway == "" && (ref.Profile == "go-kubernetes-isolated-v1" || ref.Profile == "playwright-kubernetes-isolated-v1" || ref.Profile == "go-kubernetes-discovery-v1" || ref.Profile == "playwright-kubernetes-discovery-v1") && ref.Container == "teststudio-job-"+func() string { sum := sha256.Sum256([]byte(ref.ID)); return hex.EncodeToString(sum[:20]) }()
}

func labels(ref execution.Reference) map[string]string {
	return map[string]string{workerLabel: "isolated-v1", "teststudio.io/installation": short(ref.Installation), "teststudio.io/profile": ref.Profile, "teststudio.io/execution": short(ref.ID)}
}

func (p *Provider) build(job hostedexecution.Job, seconds int) pod {
	ref, _ := json.Marshal(job.Reference)
	selectors := map[string]any{"kubernetes.io/os": "linux", "kubernetes.io/arch": p.config.Architecture}
	for key, value := range p.config.NodeSelector {
		selectors[key] = value
	}
	pulls := []any{}
	for _, name := range p.config.ImagePullSecrets {
		pulls = append(pulls, map[string]any{"name": name})
	}
	security := map[string]any{"runAsNonRoot": true, "runAsUser": 1000, "runAsGroup": 1000, "fsGroup": 1000, "fsGroupChangePolicy": "OnRootMismatch", "seccompProfile": map[string]any{"type": "RuntimeDefault"}}
	containerSecurity := map[string]any{"allowPrivilegeEscalation": false, "readOnlyRootFilesystem": true, "capabilities": map[string]any{"drop": []string{"ALL"}}}
	args := []string{"--runner=" + job.Runner, "--exchange-origin=" + job.ExchangeOrigin}
	if job.Operation == "discover" {
		args = append(args, "--operation=discover")
	}
	return pod{APIVersion: "v1", Kind: "Pod", Metadata: metadata{Name: job.Reference.Container, Namespace: p.config.Namespace, Labels: labels(job.Reference), Annotations: map[string]string{ownerAnnotation: string(ref), deadlineAnnotation: job.Ticket.Deadline.UTC().Format(time.RFC3339Nano), secondsAnnotation: strconv.Itoa(seconds), "sidecar.istio.io/inject": "false", "linkerd.io/inject": "disabled"}}, Spec: map[string]any{
		"serviceAccountName": p.config.ServiceAccount, "automountServiceAccountToken": false, "enableServiceLinks": false, "restartPolicy": "Never", "runtimeClassName": p.config.RuntimeClass, "activeDeadlineSeconds": seconds, "terminationGracePeriodSeconds": 10, "hostNetwork": false, "hostPID": false, "hostIPC": false, "shareProcessNamespace": false, "dnsPolicy": "ClusterFirst", "nodeSelector": selectors, "imagePullSecrets": pulls, "securityContext": security,
		"tolerations": []any{map[string]any{"key": "node.kubernetes.io/not-ready", "operator": "Exists", "effect": "NoExecute", "tolerationSeconds": 0}, map[string]any{"key": "node.kubernetes.io/unreachable", "operator": "Exists", "effect": "NoExecute", "tolerationSeconds": 0}},
		"containers":  []any{map[string]any{"name": "runner", "image": job.Image, "imagePullPolicy": "IfNotPresent", "command": []string{"/runner/teststudio-job"}, "args": args, "env": []any{map[string]any{"name": "TESTSTUDIO_JOB_EXECUTION", "value": job.Reference.ID}, map[string]any{"name": "TESTSTUDIO_JOB_TOKEN", "value": job.Ticket.Token}, map[string]any{"name": "TESTSTUDIO_JOB_INSTANCE", "valueFrom": map[string]any{"fieldRef": map[string]any{"apiVersion": "v1", "fieldPath": "metadata.uid"}}}}, "securityContext": containerSecurity, "terminationMessagePath": "/dev/termination-log", "terminationMessagePolicy": "File", "resources": map[string]any{"requests": map[string]string{"cpu": "2", "memory": "2Gi", "ephemeral-storage": "2Gi"}, "limits": map[string]string{"cpu": "2", "memory": "2Gi", "ephemeral-storage": "2Gi"}}, "volumeMounts": []any{map[string]any{"name": "temporary", "mountPath": "/tmp"}, map[string]any{"name": "output", "mountPath": "/output"}}}},
		"volumes":     []any{map[string]any{"name": "temporary", "emptyDir": map[string]any{"medium": "Memory", "sizeLimit": "1Gi"}}, map[string]any{"name": "output", "emptyDir": map[string]any{"sizeLimit": "1Gi"}}},
	}}
}

func normalizedSpec(value map[string]any, seconds int, stopping bool) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, hostedexecution.ErrOwnership
	}
	var result map[string]any
	if json.Unmarshal(data, &result) != nil {
		return nil, hostedexecution.ErrOwnership
	}
	if alias, ok := result["serviceAccount"]; ok && alias == result["serviceAccountName"] {
		delete(result, "serviceAccount")
	}
	if node, ok := result["nodeName"].(string); ok && len(node) > 0 && len(node) <= 253 {
		delete(result, "nodeName")
	}
	for _, key := range []string{"hostNetwork", "hostPID", "hostIPC", "shareProcessNamespace"} {
		if _, ok := result[key]; !ok {
			result[key] = false
		}
	}
	if result["hostUsers"] == true {
		delete(result, "hostUsers")
	}
	if pulls, ok := result["imagePullSecrets"].([]any); ok && len(pulls) == 0 {
		delete(result, "imagePullSecrets")
	}
	for key, want := range map[string]any{"schedulerName": "default-scheduler", "priority": float64(0), "preemptionPolicy": "PreemptLowerPriority"} {
		if result[key] == want {
			delete(result, key)
		}
	}
	if actual, ok := result["activeDeadlineSeconds"].(float64); !ok || actual < 1 || actual > float64(seconds) || !stopping && actual != float64(seconds) {
		return nil, hostedexecution.ErrOwnership
	}
	result["activeDeadlineSeconds"] = seconds
	return result, nil
}

func (p *Provider) verifyPod(value pod, record dispatch) error {
	var ref execution.Reference
	if value.Kind != "Pod" || value.APIVersion != "v1" || value.Metadata.Name != record.Reference.Container || value.Metadata.Namespace != p.config.Namespace || value.Metadata.UID == "" || value.Metadata.Version == "" || len(value.Metadata.OwnerReferences) != 0 || json.Unmarshal([]byte(value.Metadata.Annotations[ownerAnnotation]), &ref) != nil || ref != record.Reference || record.UID != "" && value.Metadata.UID != record.UID {
		return hostedexecution.ErrOwnership
	}
	for key, want := range labels(ref) {
		if value.Metadata.Labels[key] != want {
			return hostedexecution.ErrOwnership
		}
	}
	deadline, err := time.Parse(time.RFC3339Nano, value.Metadata.Annotations[deadlineAnnotation])
	if err != nil || !deadline.Equal(record.Deadline) || value.Metadata.Annotations[specAnnotation] != record.SpecHash || value.Metadata.Annotations[secondsAnnotation] != strconv.Itoa(record.ActiveSeconds) {
		return hostedexecution.ErrOwnership
	}
	normalized, err := normalizedSpec(value.Spec, record.ActiveSeconds, record.Stopping)
	if err != nil || fingerprint(normalized) != record.SpecHash {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *Provider) jobRecord(job hostedexecution.Job) (dispatch, pod, error) {
	profile := job.Runner + "-kubernetes-isolated-v1"
	if job.Operation == "discover" {
		profile = job.Runner + "-kubernetes-discovery-v1"
	}

	if !validReference(job.Reference) || job.Runner != "go" && job.Runner != "playwright" || job.Operation != "" && job.Operation != "discover" || job.Reference.Profile != profile || job.ExchangeOrigin != p.config.ExchangeOrigin || !immutable.MatchString(job.Image) || job.Ticket.Execution != job.Reference.ID || len(job.Ticket.Token) != 43 || strings.ContainsAny(job.Ticket.Token, "\r\n\x00") || !job.Ticket.Deadline.After(time.Now()) || job.Ticket.Deadline.After(time.Now().Add(2*time.Hour)) {
		return dispatch{}, pod{}, hostedexecution.ErrConfiguration
	}
	seconds := int(time.Until(job.Ticket.Deadline).Seconds()) + 1
	value := p.build(job, seconds)
	spec, err := normalizedSpec(value.Spec, seconds, false)
	if err != nil {
		return dispatch{}, pod{}, err
	}
	hash := fingerprint(spec)
	value.Metadata.Annotations[specAnnotation] = hash
	now := time.Now().UTC()
	record := dispatch{Scope: p.scope(), Reference: job.Reference, Image: job.Image, Deadline: job.Ticket.Deadline, Fingerprint: fingerprint([]any{job.Reference, job.Image, job.Runner, job.ExchangeOrigin, job.Ticket.Deadline, job.Ticket.Token}), SpecHash: hash, ActiveSeconds: seconds, CreatedAt: now, GuardUntil: now.Add(30 * time.Second)}
	return record, value, nil
}

func statusOf(value pod, record dispatch) (hostedexecution.Status, error) {
	status := hostedexecution.Status{Reference: record.Reference, Image: record.Image, Deadline: record.Deadline, Phase: "pending"}
	if len(value.Status.Containers) > 1 {
		return status, hostedexecution.ErrOwnership
	}
	var exit *int
	if len(value.Status.Containers) == 1 {
		container := value.Status.Containers[0]
		if container.Name != "runner" || container.Restarts != 0 {
			return status, hostedexecution.ErrOwnership
		}
		if container.State.Terminated != nil {
			code := container.State.Terminated.Exit
			if code < 0 || code > 255 {
				return status, hostedexecution.ErrOwnership
			}
			exit = &code
		}
	}
	switch value.Status.Phase {
	case "", "Pending":
	case "Running":
		status.Phase = "running"
	case "Succeeded":
		if exit == nil || *exit != 0 {
			return status, hostedexecution.ErrUnavailable
		}
		status.Phase = "succeeded"
		status.ExitCode = exit
	case "Failed":
		if exit == nil {
			return status, hostedexecution.ErrUnavailable
		}
		status.Phase = "failed"
		status.ExitCode = exit
		if record.Stopping {
			status.Phase = "stopped"
		} else if exit != nil && *exit == 0 {
			return status, hostedexecution.ErrOwnership
		}
	default:
		return status, hostedexecution.ErrUnavailable
	}
	return status, nil
}

func (p pod) String() string   { return fmt.Sprintf("owned Kubernetes Pod %s", p.Metadata.Name) }
func (p pod) GoString() string { return p.String() }
