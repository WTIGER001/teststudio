package azuresandbox

import (
	"context"
	"errors"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
)

func (m *ImageManager) unusedGeneration(ctx context.Context, plan *ImagePlan, job hostedexecution.Job) error {
	row, err := m.exchange.Store.Get(ctx, ImageHistoryBucket, plan.proof.Fingerprint)
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	if err != nil {
		return hostedexecution.ErrUnavailable
	}
	d, err := storage.Decode[imageImport](row)
	if err != nil || row.Project != job.Reference.Project || d.Proof != plan.proof || d.RequestedBy != job.Reference || !failedImport(d) {
		return hostedexecution.ErrOwnership
	}
	return hostedexecution.ErrRejected
}

func (m *ImageManager) load(ctx context.Context, want *ImagePlan) (imageImport, storage.Record, *ImagePlan, error) {
	row, err := m.exchange.Store.Get(ctx, ImageBucket, want.key())
	if errors.Is(err, storage.ErrNotFound) {
		err = hostedexecution.ErrNotFound
	}
	if err != nil {
		return imageImport{}, row, nil, controlError(err)
	}
	d, err := storage.Decode[imageImport](row)
	if err != nil || row.ID != want.key() || row.Project != want.proof.Image.Project || d.CreatedAt.IsZero() || !validReference(d.RequestedBy, d.Proof.Image.Runner, "discover") || hash(d.RequestedBy) != d.Proof.Generation || d.RequestedBy.Project != row.Project {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	p, err := NewImagePlan(m.config, m.identity, d.Proof.Image, d.Proof.Generation)
	if err != nil || p.proof != d.Proof || p.key() != want.key() || d.Rejected && (!d.Attempted || d.Receipt != "") || d.Blocked && d.Attempted || d.Receipt != "" && (!d.Attempted || !identifier.MatchString(d.Receipt)) || d.LastState != "" && (d.Receipt == "" || !identifier.MatchString(d.LastState)) {
		return d, row, nil, hostedexecution.ErrOwnership
	}
	return d, row, p, nil
}

func (m *ImageManager) update(ctx context.Context, plan *ImagePlan, mutate func(*imageImport) error, guards []storage.Change) (imageImport, error) {
	for range 8 {
		d, row, actual, err := m.load(ctx, plan)
		if err != nil {
			return d, err
		}
		if actual.proof != plan.proof {
			return d, hostedexecution.ErrOwnership
		}
		if err := mutate(&d); err != nil {
			return d, err
		}
		change, err := storage.Put(row.Bucket, row.ID, row.Project, row.Revision, d)
		if err != nil {
			return d, hostedexecution.ErrUnavailable
		}
		err = m.exchange.Store.Commit(ctx, append(append([]storage.Change{}, guards...), change)...)
		if errors.Is(err, storage.ErrConflict) {
			continue
		}
		return d, controlError(err)
	}
	return imageImport{}, hostedexecution.ErrUnavailable
}

func (m *ImageManager) resolve(ctx context.Context, d imageImport, plan *ImagePlan) (DiskImage, error) {
	if d.Blocked {
		return DiskImage{}, hostedexecution.ErrOwnership
	}
	if d.Rejected {
		return DiskImage{}, hostedexecution.ErrRejected
	}
	if !d.Attempted {
		return DiskImage{}, hostedexecution.ErrUnavailable
	}
	if d.Receipt != "" {
		var image DiskImage
		err := bounded(ctx, func(ctx context.Context) error { var e error; image, e = m.images.ReadImage(ctx, d.Receipt); return e })
		if err != nil {
			return DiskImage{}, hostedexecution.ErrUnavailable
		}
		if image.ID != d.Receipt {
			return DiskImage{}, hostedexecution.ErrOwnership
		}
		return m.remember(ctx, plan, image)
	}
	var items []DiskImage
	if err := bounded(ctx, func(ctx context.Context) error { var e error; items, e = m.images.Images(ctx); return e }); err != nil {
		return DiskImage{}, err
	}
	var found *DiskImage
	for _, image := range items {
		if plan.Verify(image) == nil {
			if found != nil {
				return DiskImage{}, hostedexecution.ErrOwnership
			}
			copy := image
			found = &copy
		}
	}
	if found == nil {
		return DiskImage{}, hostedexecution.ErrUnavailable
	}
	return m.remember(ctx, plan, *found)
}

func (m *ImageManager) remember(ctx context.Context, plan *ImagePlan, image DiskImage) (DiskImage, error) {
	if plan.Verify(image) != nil || !identifier.MatchString(image.State) {
		return DiskImage{}, hostedexecution.ErrOwnership
	}
	_, err := m.update(ctx, plan, func(current *imageImport) error {
		if !current.Attempted || current.Rejected || current.Blocked || current.Receipt != "" && current.Receipt != image.ID || current.LastState == "Failed" && image.State != "Failed" {
			return hostedexecution.ErrOwnership
		}
		current.Receipt = image.ID
		current.LastState = image.State
		return nil
	}, nil)
	if err != nil {
		return DiskImage{}, err
	}
	if image.State == "Failed" {
		return DiskImage{}, hostedexecution.ErrRejected
	}
	return image, nil
}
