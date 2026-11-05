package azuresandbox

import (
	"slices"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

type ImageProof struct {
	Image        hostedexecution.Image `json:"image"`
	PullIdentity string                `json:"pullIdentity"`
	Generation   string                `json:"generation"`
	Fingerprint  string                `json:"fingerprint"`
}

type ImagePlan struct {
	config Config
	proof  ImageProof
}

func NewImagePlan(c Config, identity string, inspected hostedexecution.Image, generation string) (*ImagePlan, error) {
	profile, err := ExecutionProfile(c)
	if err != nil || !c.validPullIdentity(identity) || !checksum.MatchString(generation) || !c.validImage(inspected.Reference) || !domain.ValidID(inspected.Project) || inspected.Runner != "go" && inspected.Runner != "playwright" || inspected.Runtime == "" || len(inspected.Runtime) > 256 || inspected.OS != "linux" || inspected.Architecture != "amd64" || inspected.Supervisor != "worker-exchange/v1" || inspected.Sandbox != hostedexecution.SandboxSupervisor || inspected.TaskStorage != hostedexecution.NonRootTaskStorage || inspected.Discovery != "worker-exchange/v1" || !validSourceDigest(inspected.SourceDigest) || inspected.ExecutionProfile != "" && inspected.ExecutionProfile != profile {
		return nil, hostedexecution.ErrConfiguration
	}
	inspected.ExecutionProfile = profile
	p := &ImagePlan{config: c, proof: ImageProof{Image: inspected, PullIdentity: identity, Generation: generation}}
	p.proof.Fingerprint = hash([]any{"azure-sandbox-image/v1", c, APIVersion, TokenScope, p.proof})
	return p, nil
}

func (p *ImagePlan) valid() error {
	if p == nil {
		return hostedexecution.ErrOwnership
	}
	restored, err := NewImagePlan(p.config, p.proof.PullIdentity, p.proof.Image, p.proof.Generation)
	if err != nil || restored.proof != p.proof {
		return hostedexecution.ErrOwnership
	}
	return nil
}

func (p *ImagePlan) key() string {
	return hash([]any{"azure-sandbox-image-cache/v1", p.config, p.proof.PullIdentity, p.proof.Image})
}

func (p *ImagePlan) definition() map[string]any {
	return map[string]any{
		"image":  map[string]any{"base": p.proof.Image.Reference, "entrypoint": []string{"/runner/teststudio-job"}, "cmd": []string{}},
		"labels": map[string]string{"name": "teststudio-" + p.proof.Fingerprint[:24], "teststudio-spec": p.proof.Fingerprint, "teststudio-scope": hash(p.config), "teststudio-project": p.proof.Image.Project, "teststudio-source": p.proof.Image.SourceDigest, "teststudio-owner": p.proof.Generation},
	}
}

func (p *ImagePlan) Verify(image DiskImage) error {
	if p.valid() != nil {
		return hostedexecution.ErrOwnership
	}
	want := p.definition()
	want["dockerfileContent"] = ""
	if !identifier.MatchString(image.ID) || image.Base != p.proof.Image.Reference || !slices.Equal(image.Entrypoint, []string{"/runner/teststudio-job"}) || len(image.Command) != 0 || image.DefinitionHash != hash(want) {
		return hostedexecution.ErrOwnership
	}
	return nil
}
