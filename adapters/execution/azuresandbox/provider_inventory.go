package azuresandbox

import (
	"context"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) Inventory(ctx context.Context, installation, profile string) ([]execution.Reference, error) {
	if p.closed.Load() {
		return nil, hostedexecution.ErrUnavailable
	}
	if !validInventoryScope(installation, profile) {
		return nil, hostedexecution.ErrConfiguration
	}
	operation := ""
	if strings.HasSuffix(profile, "-discovery-v1") {
		operation = "discover"
	}
	if !p.permits(operation) {
		return nil, hostedexecution.ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	refs := []execution.Reference{}
	active := map[string]execution.Reference{}
	cursor := ""
	complete := false
	for range 100 {
		page, err := p.controller.store.List(ctx, IntentBucket, "", cursor, 100)
		if err != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, row := range page.Items {
			d, err := storage.Decode[launchIntent](row)
			if err != nil {
				return nil, hostedexecution.ErrOwnership
			}
			ref := d.Proof.Reference
			if ref.Installation != installation || ref.Profile != profile {
				continue
			}
			if _, _, err := p.loadIntent(ctx, ref); err != nil {
				return nil, err
			}
			if !d.Removed {
				refs = append(refs, ref)
				active[ref.ID] = ref
			}
		}
		cursor = page.Cursor
		if cursor == "" {
			complete = true
			break
		}
	}
	if !complete {
		return nil, hostedexecution.ErrUnavailable
	}
	controlled, err := p.controller.Inventory(ctx, installation, profile)
	if err != nil {
		return nil, err
	}
	for _, ref := range controlled {
		if active[ref.ID] != ref {
			return nil, hostedexecution.ErrOwnership
		}
	}
	return refs, nil
}
