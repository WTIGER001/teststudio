package localcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

type containerInfo struct {
	ID     string `json:"Id"`
	Config struct {
		Labels map[string]string `json:"Labels"`
		Image  string            `json:"Image"`
	} `json:"Config"`
	State struct {
		Running  bool `json:"Running"`
		ExitCode int  `json:"ExitCode"`
		Health   struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	Mounts []struct{ Type, Name string } `json:"Mounts"`
}
type volumeInfo struct {
	Name   string
	Labels map[string]string
}

func (e *Executor) Reference(ref execution.Reference) execution.Reference {
	prefix := "pw"
	if e.runner() == "go" {
		prefix = "go"
	}
	if e.Environment {
		prefix += "-env"
		ref.Gateway = "teststudio-" + strings.TrimSuffix(prefix, "-env") + "-gateway-" + ref.ID
	}
	ref.Container = "teststudio-" + prefix + "-" + ref.ID
	ref.Volume = "teststudio-" + prefix + "-output-" + ref.ID
	ref.Profile = e.Profile()
	return ref
}
func labels(ref execution.Reference) map[string]string {
	return map[string]string{"teststudio.worker": ref.Profile, "teststudio.installation": ref.Installation, "teststudio.execution": ref.ID, "teststudio.run": ref.Run, "teststudio.project": ref.Project, "teststudio.test": ref.Test, "teststudio.attempt": fmt.Sprint(ref.Attempt), "teststudio.owner": ref.Worker}
}
func ownedLabels(ref execution.Reference, actual map[string]string) bool {
	for key, value := range labels(ref) {
		if actual[key] != value {
			return false
		}
	}
	return true
}
func (e *Executor) output(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, e.binary(), args...)
	var output boundedOutput
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if output.Truncated {
		return nil, errors.New("Docker response exceeds its output limit")
	}
	if err != nil {
		return output.Bytes(), fmt.Errorf("Docker %s: %w: %s", args[0], err, strings.TrimSpace(string(output.Bytes())))
	}
	return output.Bytes(), nil
}
func absent(err error, kind string) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no such "+kind)
}
func (e *Executor) container(ctx context.Context, ref execution.Reference) (containerInfo, bool, error) {
	data, err := e.output(ctx, "container", "inspect", ref.Container)
	if absent(err, "container") || absent(err, "object") {
		return containerInfo{}, false, nil
	}
	if err != nil {
		return containerInfo{}, false, err
	}
	var items []containerInfo
	if err := json.Unmarshal(data, &items); err != nil || len(items) != 1 {
		return containerInfo{}, false, errors.New("Docker container metadata is invalid")
	}
	if !ownedLabels(ref, items[0].Config.Labels) {
		return containerInfo{}, false, errors.New("worker container ownership does not match its durable reference")
	}
	return items[0], true, nil
}
func (e *Executor) volume(ctx context.Context, ref execution.Reference) (bool, error) {
	data, err := e.output(ctx, "volume", "inspect", ref.Volume)
	if absent(err, "volume") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var items []volumeInfo
	if err := json.Unmarshal(data, &items); err != nil || len(items) != 1 {
		return false, errors.New("Docker volume metadata is invalid")
	}
	if !ownedLabels(ref, items[0].Labels) {
		return false, errors.New("worker volume ownership does not match its durable reference")
	}
	return true, nil
}
func (e *Executor) Cleanup(ctx context.Context, ref execution.Reference) error {
	if ref.Profile != e.Profile() {
		return errors.New("worker cleanup profile does not match its durable reference")
	}
	container, exists, err := e.container(ctx, ref)
	if err != nil {
		return err
	}
	volume, err := e.volume(ctx, ref)
	if err != nil {
		return err
	}
	if exists {
		if _, err := e.output(ctx, "rm", "--force", "--volumes", container.ID); err != nil && !absent(err, "container") {
			return err
		}
	}
	if ref.Gateway != "" {
		gatewayRef := ref
		gatewayRef.Container = ref.Gateway
		gateway, exists, err := e.container(ctx, gatewayRef)
		if err != nil {
			return err
		}
		if exists {
			if _, err := e.output(ctx, "rm", "--force", "--volumes", gateway.ID); err != nil && !absent(err, "container") {
				return err
			}
		}
	}
	if volume {
		if _, err := e.output(ctx, "volume", "rm", ref.Volume); err != nil && !absent(err, "volume") {
			return err
		}
	}
	return nil
}
func (e *Executor) prepareVolume(ctx context.Context, ref execution.Reference) error {
	exists, err := e.volume(ctx, ref)
	if err != nil || exists {
		return err
	}
	args := []string{"volume", "create"}
	for key, value := range labels(ref) {
		args = append(args, "--label", key+"="+value)
	}
	args = append(args, ref.Volume)
	_, err = e.output(ctx, args...)
	if err != nil {
		return err
	}
	_, err = e.volume(ctx, ref)
	return err
}
func (e *Executor) Recover(ctx context.Context, ref execution.Reference, input execution.CodedInput) execution.CodedReport {
	started := time.Now().UTC()
	unknown := func(err error) execution.CodedReport {
		return execution.CodedReport{RecoveryError: err, Result: execution.Result{TestID: ref.Test, DurationUnavailable: true, State: "unknown", FailureKind: "worker-lost", Error: err.Error(), StartedAt: started, FinishedAt: time.Now().UTC()}, EvidenceErrors: []string{"Worker recovery cannot prove a complete observation."}}
	}
	if ref.Profile != e.Profile() {
		return unknown(errors.New("worker recovery profile does not match its durable reference"))
	}
	info, exists, err := e.container(ctx, ref)
	if err != nil {
		return unknown(err)
	}
	if !exists {
		report := unknown(errors.New("worker container is no longer available"))
		report.RecoveryError = nil
		return report
	}
	if info.Config.Image != input.Artifact.Image {
		return unknown(errors.New("worker image does not match its immutable release"))
	}
	if info.State.Running {
		if _, err := e.output(ctx, "stop", "--time=5", info.ID); err != nil {
			return unknown(err)
		}
		info, _, err = e.container(ctx, ref)
		if err != nil {
			return unknown(err)
		}
	}
	return e.collectReport(ctx, ref.Container, input, func() error {
		if info.State.ExitCode != 0 {
			return fmt.Errorf("worker exited with status %d", info.State.ExitCode)
		}
		return nil
	}(), true)
}

func parseResourceLabels(source string) map[string]string {
	labels := map[string]string{}
	for _, part := range strings.Split(source, ",") {
		key, value, ok := strings.Cut(part, "=")
		if ok {
			labels[key] = value
		}
	}
	return labels
}
func (e *Executor) Inventory(ctx context.Context, installation string) ([]execution.Reference, error) {
	found := map[string]execution.Reference{}
	for _, kind := range []string{"container", "volume"} {
		args := []string{"ps", "--all"}
		if kind == "volume" {
			args = []string{"volume", "ls"}
		}
		args = append(args, "--filter", "label=teststudio.worker="+e.Profile(), "--filter", "label=teststudio.installation="+installation, "--format", "{{json .}}")
		data, err := e.output(ctx, args...)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var item struct{ Names, Name, Labels string }
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				return nil, err
			}
			values := parseResourceLabels(item.Labels)
			attempt, _ := strconv.Atoi(values["teststudio.attempt"])
			ref := e.Reference(execution.Reference{ID: values["teststudio.execution"], Installation: values["teststudio.installation"], Project: values["teststudio.project"], Run: values["teststudio.run"], Test: values["teststudio.test"], Attempt: attempt, Worker: values["teststudio.owner"]})
			if !domain.ValidID(ref.ID) || !domain.ValidID(ref.Run) || !domain.ValidID(ref.Project) || !domain.ValidID(ref.Test) || ref.Attempt < 1 || ref.Attempt > 3 || ref.Installation != installation || !ownedLabels(ref, values) {
				return nil, errors.New("managed worker inventory contains invalid ownership metadata")
			}
			name := item.Names
			if kind == "volume" {
				name = item.Name
			}
			if name != ref.Container && name != ref.Volume && name != ref.Gateway {
				return nil, errors.New("managed worker resource name does not match its identity")
			}
			found[ref.ID] = ref
		}
	}
	refs := []execution.Reference{}
	for _, ref := range found {
		refs = append(refs, ref)
	}
	slices.SortFunc(refs, func(a, b execution.Reference) int { return strings.Compare(a.ID, b.ID) })
	return refs, nil
}
