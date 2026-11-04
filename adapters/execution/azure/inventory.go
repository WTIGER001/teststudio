package azureworker

import (
	"context"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (p *Provider) Inventory(ctx context.Context, installation, profile string) (refs []execution.Reference, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	validProfile := false
	for _, runner := range []string{"go", "playwright"} {
		validProfile = validProfile || profile == runner+"-"+Location+"-isolated-v1" || profile == runner+"-"+Location+"-discovery-v1"
	}
	if !domain.ValidID(installation) || !validProfile {
		return nil, hostedexecution.ErrConfiguration
	}
	cursor := ""
	for {
		page, err := p.store.List(ctx, Bucket, "", cursor, 100)
		if err != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, row := range page.Items {
			d, err := storage.Decode[dispatch](row)
			if err != nil {
				return nil, hostedexecution.ErrOwnership
			}
			ref := d.Proof.Reference
			if ref.Installation != installation || ref.Profile != profile || d.Removed {
				continue
			}
			if _, err := Restore(p.config, d.Proof); err != nil {
				continue
			}
			if _, _, _, err := p.load(ctx, ref); err != nil {
				return nil, err
			}
			refs = append(refs, ref)
		}
		cursor = page.Cursor
		if cursor == "" {
			return refs, nil
		}
	}
}
