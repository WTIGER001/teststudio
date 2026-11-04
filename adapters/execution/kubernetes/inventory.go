package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (p *Provider) adopt(ctx context.Context, ref execution.Reference) (dispatch, error) {
	var value pod
	if err := p.request(ctx, http.MethodGet, p.podPath(ref.Container), nil, nil, &value); errors.Is(err, errAbsent) {
		return dispatch{}, hostedexecution.ErrNotFound
	} else if err != nil {
		return dispatch{}, hostedexecution.ErrUnavailable
	}
	entry, _, err := (workerstate.Ledger{Store: p.store}).Get(ctx, ref.Run, ref.ID)
	if err != nil || entry.Reference != ref {
		return dispatch{}, hostedexecution.ErrOwnership
	}
	data, _ := json.Marshal(value.Spec["containers"])
	var containers []struct {
		Image string   `json:"image"`
		Args  []string `json:"args"`
		Env   []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"env"`
	}
	if json.Unmarshal(data, &containers) != nil || len(containers) != 1 {
		return dispatch{}, hostedexecution.ErrOwnership
	}
	token := ""
	for _, env := range containers[0].Env {
		if env.Name == "TESTSTUDIO_JOB_TOKEN" {
			token = env.Value
		}
	}
	if len(token) != 43 || !immutable.MatchString(containers[0].Image) {
		return dispatch{}, hostedexecution.ErrOwnership
	}
	deadline, err := time.Parse(time.RFC3339Nano, value.Metadata.Annotations[deadlineAnnotation])
	if err != nil {
		return dispatch{}, hostedexecution.ErrOwnership
	}
	seconds, err := strconv.Atoi(value.Metadata.Annotations[secondsAnnotation])
	if err != nil || seconds < 1 || seconds > 7201 {
		return dispatch{}, hostedexecution.ErrOwnership
	}
	runner := strings.TrimSuffix(ref.Profile, "-kubernetes-isolated-v1")
	job := hostedexecution.Job{Reference: ref, Image: containers[0].Image, Runner: runner, ExchangeOrigin: p.config.ExchangeOrigin, Ticket: workerexchange.Ticket{Execution: ref.ID, Token: token, Deadline: deadline}}
	expected := p.build(job, seconds)
	normalized, err := normalizedSpec(expected.Spec, seconds, false)
	if err != nil {
		return dispatch{}, err
	}
	record := dispatch{Scope: p.scope(), Reference: ref, Image: job.Image, Deadline: deadline, SpecHash: fingerprint(normalized), Fingerprint: fingerprint([]any{job.Reference, job.Image, job.Runner, job.ExchangeOrigin, job.Ticket.Deadline, job.Ticket.Token}), ActiveSeconds: seconds, CreatedAt: time.Now().UTC(), GuardUntil: time.Now().UTC().Add(30 * time.Second), UID: value.Metadata.UID}
	if err := p.verifyPod(value, record); err != nil {
		return record, err
	}
	record, _, err = p.reserve(ctx, record)
	return record, err
}

func (p *Provider) Inventory(ctx context.Context, installation, profile string) ([]execution.Reference, error) {
	if !domain.ValidID(installation) || (profile != "go-kubernetes-isolated-v1" && profile != "playwright-kubernetes-isolated-v1") {
		return nil, hostedexecution.ErrConfiguration
	}
	found := map[string]execution.Reference{}
	continuation := ""
	for page := 0; page < 100; page++ {
		var list struct {
			Items    []pod `json:"items"`
			Metadata struct {
				Continue string `json:"continue"`
			} `json:"metadata"`
		}
		query := url.Values{"labelSelector": {workerLabel + "=isolated-v1,teststudio.io/installation=" + short(installation) + ",teststudio.io/profile=" + profile}, "limit": {"100"}}
		if continuation != "" {
			query.Set("continue", continuation)
		}
		if err := p.request(ctx, http.MethodGet, p.podsPath(), query, nil, &list); err != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, value := range list.Items {
			var ref execution.Reference
			if len(value.Metadata.Annotations[ownerAnnotation]) > 4096 || json.Unmarshal([]byte(value.Metadata.Annotations[ownerAnnotation]), &ref) != nil || !validReference(ref) || ref.Installation != installation || ref.Profile != profile || value.Metadata.Name != ref.Container || value.Metadata.Namespace != p.config.Namespace || value.Metadata.UID == "" || len(value.Metadata.OwnerReferences) != 0 {
				return nil, hostedexecution.ErrOwnership
			}
			for key, want := range labels(ref) {
				if value.Metadata.Labels[key] != want {
					return nil, hostedexecution.ErrOwnership
				}
			}
			if record, _, err := p.load(ctx, ref); err == nil {
				if err := p.verifyPod(value, record); err != nil {
					return nil, err
				}
			} else if !errors.Is(err, storage.ErrNotFound) {
				return nil, err
			}
			if _, ok := found[ref.ID]; ok {
				return nil, hostedexecution.ErrOwnership
			}
			found[ref.ID] = ref
		}
		if list.Metadata.Continue == "" {
			break
		}
		if page == 99 || list.Metadata.Continue == continuation {
			return nil, hostedexecution.ErrUnavailable
		}
		continuation = list.Metadata.Continue
	}
	cursor := ""
	for {
		page, err := p.store.List(ctx, Bucket, "", cursor, 100)
		if err != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, row := range page.Items {
			record, err := storage.Decode[dispatch](row)
			if err != nil {
				return nil, hostedexecution.ErrUnavailable
			}
			ref := record.Reference
			if record.Removed || ref.Installation != installation || ref.Profile != profile {
				continue
			}
			if record.Scope != p.scope() {
				return nil, hostedexecution.ErrOwnership
			}
			if !validReference(ref) || row.Project != ref.Project {
				return nil, hostedexecution.ErrOwnership
			}
			found[ref.ID] = ref
		}
		if page.Cursor == "" {
			break
		}
		cursor = page.Cursor
	}
	refs := make([]execution.Reference, 0, len(found))
	for _, ref := range found {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(a, b int) bool { return refs[a].ID < refs[b].ID })
	return refs, nil
}
