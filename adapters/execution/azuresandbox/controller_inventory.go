package azuresandbox

import (
	"context"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (c *Controller) Inventory(ctx context.Context, installation, profile string) (refs []execution.Reference, resultErr error) {
	defer func() { resultErr = controlError(resultErr) }()
	validProfile := false
	for _, runner := range []string{"go", "playwright"} {
		validProfile = validProfile || profile == runner+"-"+Location+"-isolated-v1" || profile == runner+"-"+Location+"-discovery-v1"
	}
	if !domain.ValidID(installation) || !validProfile {
		return nil, hostedexecution.ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cursor := ""
	refs = []execution.Reference{}
	for pages := 0; pages < 100; pages++ {
		page, err := c.store.List(ctx, DispatchBucket, "", cursor, 100)
		if err != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		for _, row := range page.Items {
			d, err := storage.Decode[dispatch](row)
			if err != nil {
				return nil, hostedexecution.ErrOwnership
			}
			ref := d.Proof.Reference
			if ref.Installation != installation || ref.Profile != profile {
				continue
			}
			if _, _, _, err := c.load(ctx, ref); err != nil {
				return nil, err
			}
			if !d.Removed {
				refs = append(refs, ref)
			}
		}
		cursor = page.Cursor
		if cursor == "" {
			return refs, nil
		}
	}
	return nil, hostedexecution.ErrUnavailable
}
