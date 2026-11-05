package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerstate"
)

func (p *Provider) networkInventory(ctx context.Context, installation, profile string) ([]execution.Reference, error) {
	found := map[string]execution.Reference{}
	seen := map[string]bool{}
	for _, kind := range []string{"Pod", "Service", "NetworkPolicy"} {
		continuation := ""
		for page := 0; page < 100; page++ {
			var list struct {
				Items    []networkObject `json:"items"`
				Metadata struct {
					Continue string `json:"continue"`
				} `json:"metadata"`
			}
			query := url.Values{"labelSelector": {"teststudio.io/installation=" + short(installation) + ",teststudio.io/profile=" + profile}, "limit": {"100"}}
			if continuation != "" {
				query.Set("continue", continuation)
			}
			if err := p.request(ctx, http.MethodGet, p.networkPath(kind, ""), query, nil, &list); err != nil {
				return nil, hostedexecution.ErrUnavailable
			}
			for _, value := range list.Items {
				ref, err := p.inventoryNetworkObject(ctx, kind, value, installation, profile)
				if err != nil {
					return nil, err
				}
				key := kind + "/" + value.Metadata.Name
				if seen[key] {
					return nil, hostedexecution.ErrOwnership
				}
				seen[key] = true
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
	}
	return p.inventoryRecords(ctx, installation, profile, found)
}

func (p *Provider) inventoryNetworkObject(ctx context.Context, kind string, value networkObject, installation, profile string) (execution.Reference, error) {
	var ref execution.Reference
	if len(value.Metadata.Annotations[ownerAnnotation]) > 4096 || json.Unmarshal([]byte(value.Metadata.Annotations[ownerAnnotation]), &ref) != nil || !validReference(ref) || ref.Gateway == "" || ref.Installation != installation || ref.Profile != profile || value.Kind != kind || value.Metadata.Namespace != p.config.Namespace || value.Metadata.UID == "" || len(value.Metadata.OwnerReferences) != 0 {
		return ref, hostedexecution.ErrOwnership
	}
	gateway := value.Metadata.Name == ref.Gateway
	if !gateway && (value.Metadata.Name != ref.Container || kind != "Pod" && kind != "NetworkPolicy") {
		return ref, hostedexecution.ErrOwnership
	}
	for key, want := range networkLabels(ref, gateway) {
		if value.Metadata.Labels[key] != want {
			return ref, hostedexecution.ErrOwnership
		}
	}
	record, _, err := p.load(ctx, ref)
	if errors.Is(err, storage.ErrNotFound) {
		entry, _, lookupErr := (workerstate.Ledger{Store: p.store}).Get(ctx, ref.Run, ref.ID)
		if lookupErr != nil || entry.Reference != ref {
			return ref, hostedexecution.ErrOwnership
		}
		return ref, nil
	}
	if err != nil {
		return ref, err
	}
	if kind == "Pod" && !gateway {
		var worker pod
		worker.APIVersion, worker.Kind, worker.Metadata, worker.Spec = value.APIVersion, value.Kind, value.Metadata, value.Spec
		err = p.verifyPod(worker, record)
	} else {
		err = p.verifyNetworkObject(value, record)
	}
	return ref, err
}
