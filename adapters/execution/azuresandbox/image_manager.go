package azuresandbox

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
	"github.com/wtiger001/teststudio/internal/storage"
	"github.com/wtiger001/teststudio/internal/workerexchange"
)

const ImageBucket = "azure-sandbox-images"
const ImageHistoryBucket = "azure-sandbox-image-history"

type ImageControl interface {
	ImportImage(context.Context, *ImagePlan) (DiskImage, error)
	ReadImage(context.Context, string) (DiskImage, error)
	Images(context.Context) ([]DiskImage, error)
}

type ImageAdmission interface {
	BeforeImport(context.Context, Config, string) error
}

type ImageManagerOptions struct {
	Config       Config
	PullIdentity string
	Images       ImageControl
	Exchange     *workerexchange.Service
	Admission    ImageAdmission
	AllowImport  bool
	Now          func() time.Time
}

type ImageManager struct {
	config    Config
	identity  string
	images    ImageControl
	exchange  *workerexchange.Service
	admission ImageAdmission
	writer    bool
	now       func() time.Time
}

type imageImport struct {
	Proof       ImageProof          `json:"proof"`
	RequestedBy execution.Reference `json:"requestedBy"`
	CreatedAt   time.Time           `json:"createdAt"`
	Attempted   bool                `json:"attempted"`
	Rejected    bool                `json:"rejected"`
	Blocked     bool                `json:"blocked"`
	Receipt     string              `json:"receipt,omitempty"`
	LastState   string              `json:"lastState,omitempty"`
}

func NewImageManager(o ImageManagerOptions) (*ImageManager, error) {
	if Validate(o.Config) != nil || !o.Config.validPullIdentity(o.PullIdentity) || o.Images == nil || o.Exchange == nil || o.Exchange.Store == nil || o.AllowImport && (o.Exchange.Authorize == nil || o.Admission == nil) {
		return nil, hostedexecution.ErrConfiguration
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	return &ImageManager{config: o.Config, identity: o.PullIdentity, images: o.Images, exchange: o.Exchange, admission: o.Admission, writer: o.AllowImport, now: now}, nil
}

func (m *ImageManager) Lookup(ctx context.Context, inspected hostedexecution.Image) (DiskImage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	plan, err := NewImagePlan(m.config, m.identity, inspected, hash("lookup"))
	if err != nil {
		return DiskImage{}, err
	}
	d, _, actual, err := m.load(ctx, plan)
	if err != nil {
		return DiskImage{}, err
	}
	return m.resolve(ctx, d, actual)
}

func (m *ImageManager) Prepare(ctx context.Context, job hostedexecution.Job, inspected hostedexecution.Image) (DiskImage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if !m.writer {
		return DiskImage{}, hostedexecution.ErrConfiguration
	}
	plan, err := NewImagePlan(m.config, m.identity, inspected, hash(job.Reference))
	if err != nil {
		return DiskImage{}, err
	}
	guards, err := m.authorize(ctx, job, plan)
	if err != nil {
		return DiskImage{}, err
	}
	d, row, actual, err := m.load(ctx, plan)
	if err != nil && err != hostedexecution.ErrNotFound {
		return DiskImage{}, err
	}
	if err == hostedexecution.ErrNotFound || failedImport(d) && d.RequestedBy != job.Reference {
		if err := m.unusedGeneration(ctx, plan, job); err != nil {
			return DiskImage{}, err
		}
		next := imageImport{Proof: plan.proof, RequestedBy: job.Reference, CreatedAt: m.now().UTC()}
		changes := append([]storage.Change{}, guards...)
		if err == nil {
			archive, e := storage.Put(ImageHistoryBucket, d.Proof.Fingerprint, row.Project, 0, d)
			if e != nil {
				return DiskImage{}, hostedexecution.ErrUnavailable
			}
			changes = append(changes, archive)
		}
		put, e := storage.Put(ImageBucket, plan.key(), inspected.Project, row.Revision, next)
		if e != nil {
			return DiskImage{}, hostedexecution.ErrUnavailable
		}
		if e = m.exchange.Store.Commit(ctx, append(changes, put)...); e != nil && !errors.Is(e, storage.ErrConflict) {
			return DiskImage{}, hostedexecution.ErrUnavailable
		}
		d, _, actual, err = m.load(ctx, plan)
		if err != nil {
			return DiskImage{}, err
		}
	}
	if d.Attempted || d.Blocked {
		return m.resolve(ctx, d, actual)
	}
	if err := bounded(ctx, func(ctx context.Context) error { return m.admission.BeforeImport(ctx, m.config, m.identity) }); err != nil {
		return DiskImage{}, err
	}
	var prior []DiskImage
	if err := bounded(ctx, func(ctx context.Context) error { var e error; prior, e = m.images.Images(ctx); return e }); err != nil {
		return DiskImage{}, err
	}
	for _, image := range prior {
		if actual.Verify(image) == nil {
			_, err = m.update(ctx, actual, func(current *imageImport) error {
				if current.Attempted {
					return hostedexecution.ErrUnavailable
				}
				current.Blocked = true
				return nil
			}, nil)
			if err != nil {
				return DiskImage{}, err
			}
			return DiskImage{}, hostedexecution.ErrOwnership
		}
	}
	guards, err = m.authorize(ctx, job, actual)
	if err != nil {
		return DiskImage{}, err
	}
	if _, err = m.update(ctx, actual, func(current *imageImport) error {
		if current.Attempted || current.Blocked {
			return hostedexecution.ErrUnavailable
		}
		current.Attempted = true
		return nil
	}, guards); err != nil {
		return DiskImage{}, err
	}
	var image DiskImage
	err = bounded(ctx, func(ctx context.Context) error { var e error; image, e = m.images.ImportImage(ctx, actual); return e })
	if err == hostedexecution.ErrRejected {
		_, saveErr := m.update(ctx, actual, func(current *imageImport) error {
			if current.Receipt != "" {
				return hostedexecution.ErrOwnership
			}
			current.Rejected = true
			return nil
		}, nil)
		if saveErr != nil {
			return DiskImage{}, saveErr
		}
	}
	if err != nil {
		return DiskImage{}, err
	}
	return m.remember(ctx, actual, image)
}

func failedImport(d imageImport) bool { return d.Rejected || d.LastState == "Failed" }

var _ ImageControl = (*Client)(nil)
