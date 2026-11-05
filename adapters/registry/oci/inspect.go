package oci

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func validDescriptor(value descriptor, limit int64) bool {
	return digest.MatchString(value.Digest) && value.Size > 0 && value.Size <= limit && len(value.URLs) == 0
}
func manifestType(value string) bool { return value == ociManifest || value == dockerManifest }
func indexType(value string) bool    { return value == ociIndex || value == dockerIndex }

func (c *Client) manifest(ctx context.Context, repo, hash string, auth *authorization) (manifest, int, error) {
	data, contentType, err := c.fetch(ctx, repo, "manifests", hash, strings.Join([]string{ociManifest, dockerManifest, ociIndex, dockerIndex}, ", "), auth)
	if err != nil {
		return manifest{}, 0, err
	}
	var value manifest
	if decode(data, &value) != nil || value.Schema != 2 || value.MediaType != "" && value.MediaType != contentType || value.ArtifactType != "" || value.Subject != nil || (!manifestType(contentType) && !indexType(contentType)) {
		return value, 0, ErrInvalid
	}
	value.MediaType = contentType
	return value, len(data), nil
}

func (c *Client) InspectImage(ctx context.Context, reference string) (hostedexecution.Image, error) {
	result, err := c.Inspect(ctx, reference)
	return result.Image, err
}

func (c *Client) Inspect(ctx context.Context, reference string) (Metadata, error) {
	return c.inspect(ctx, reference, true)
}

func (c *Client) inspect(ctx context.Context, reference string, requireOwnership bool) (Metadata, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	repo, hash, err := c.selected(reference)
	if err != nil {
		return Metadata{}, err
	}
	auth, err := credential(ctx, c.credentials)
	if err != nil {
		return Metadata{}, err
	}
	session := &authorization{base: auth, current: auth}
	value, _, err := c.manifest(ctx, repo, hash, session)
	if err != nil {
		return Metadata{}, err
	}
	if indexType(value.MediaType) {
		if len(value.Manifests) < 1 || len(value.Manifests) > 64 {
			return Metadata{}, ErrInvalid
		}
		var selected *descriptor
		for _, item := range value.Manifests {
			p := item.Platform
			if p == nil || p.OS != "linux" || p.Architecture != c.architecture {
				continue
			}
			if selected != nil || !manifestType(item.MediaType) || !validDescriptor(item, metadataLimit) || len(p.Features) > 0 || (p.Variant != "" && (c.architecture != "arm64" || p.Variant != "v8")) {
				return Metadata{}, ErrInvalid
			}
			copy := item
			selected = &copy
		}
		if selected == nil {
			return Metadata{}, ErrInvalid
		}
		hash = selected.Digest
		var size int
		value, size, err = c.manifest(ctx, repo, hash, session)
		if err != nil {
			return Metadata{}, err
		}
		if int64(size) != selected.Size || value.MediaType != selected.MediaType {
			return Metadata{}, ErrInvalid
		}
	}
	if !manifestType(value.MediaType) || !validDescriptor(value.Config, metadataLimit) || !slices.Contains([]string{"application/vnd.oci.image.config.v1+json", "application/vnd.docker.container.image.v1+json"}, value.Config.MediaType) || len(value.Layers) < 1 || len(value.Layers) > 128 || len(value.Manifests) > 0 {
		return Metadata{}, ErrInvalid
	}
	var total int64
	for _, layer := range value.Layers {
		if !validDescriptor(layer, 8<<30) || !slices.Contains([]string{"application/vnd.oci.image.layer.v1.tar", "application/vnd.oci.image.layer.v1.tar+gzip", "application/vnd.oci.image.layer.v1.tar+zstd", "application/vnd.docker.image.rootfs.diff.tar.gzip"}, layer.MediaType) {
			return Metadata{}, ErrInvalid
		}
		total += layer.Size
		if total > 16<<30 {
			return Metadata{}, ErrInvalid
		}
	}
	data, _, err := c.fetch(ctx, repo, "blobs", value.Config.Digest, "application/octet-stream", session)
	if err != nil {
		return Metadata{}, err
	}
	if int64(len(data)) != value.Config.Size {
		return Metadata{}, ErrInvalid
	}
	var config configuration
	if decode(data, &config) != nil || config.OS != "linux" || config.Architecture != c.architecture || (config.Variant != "" && (c.architecture != "arm64" || config.Variant != "v8")) || config.Config.User != "1000:1000" || config.Config.WorkingDir != "/suite" || len(config.Config.OnBuild) > 0 || config.RootFS.Type != "layers" || len(config.RootFS.DiffIDs) != len(value.Layers) {
		return Metadata{}, ErrInvalid
	}
	for _, id := range config.RootFS.DiffIDs {
		if !digest.MatchString(id) {
			return Metadata{}, ErrInvalid
		}
	}
	labels := config.Config.Labels
	storage := labels["teststudio.task-storage"]
	if storage != "" {
		if storage != hostedexecution.NonRootTaskStorage || len(config.Config.Volumes) != 2 {
			return Metadata{}, ErrInvalid
		}
		for _, path := range []string{"/tmp", "/output"} {
			value, exists := config.Config.Volumes[path]
			var attributes map[string]json.RawMessage
			if !exists || json.Unmarshal(value, &attributes) != nil || attributes == nil || len(attributes) != 0 {
				return Metadata{}, ErrInvalid
			}
		}
	}
	runner := ""
	if strings.HasPrefix(labels["teststudio.runtime"], "go") {
		runner = "go"
	} else if strings.HasPrefix(labels["teststudio.runtime"], "playwright") {
		runner = "playwright"
	}
	entry := []string{"/runner/teststudio-go"}
	if runner == "playwright" {
		entry = []string{"node", "/runner/run.mjs"}
	}
	if runner == "" || !runtimeLabel.MatchString(labels["teststudio.runtime"]) || !slices.Equal(config.Config.Entrypoint, entry) || labels["teststudio.job-supervisor"] != "worker-exchange/v1" || labels["teststudio.runtime-kind"] != "source-base/v1" || labels["teststudio.work-item-bytes"] != "4194304" || !digest.MatchString(labels["teststudio.job-runtime-base"]) {
		return Metadata{}, ErrInvalid
	}
	if requireOwnership && (!domain.ValidID(labels["teststudio.project"]) || !domain.ValidID(labels["teststudio.installation"]) || !domain.ValidID(labels["teststudio.preparation"]) || !digest.MatchString(labels["teststudio.source"])) {
		return Metadata{}, ErrInvalid
	}
	return Metadata{template: &runtimeTemplate{client: c, repository: repo, manifest: value, configuration: data, labels: labels}, Image: hostedexecution.Image{TaskStorage: storage, Discovery: labels["teststudio.discovery-supervisor"], Reference: reference, Runner: runner, Runtime: labels["teststudio.runtime"], OS: config.OS, Architecture: config.Architecture, Supervisor: labels["teststudio.job-supervisor"], Sandbox: labels["teststudio.sandbox-supervisor"], Project: labels["teststudio.project"], SourceDigest: labels["teststudio.source"]}, Manifest: strings.TrimPrefix(c.origin, "https://") + "/" + repo + "@" + hash, Configuration: value.Config.Digest, Project: labels["teststudio.project"], Installation: labels["teststudio.installation"], Preparation: labels["teststudio.preparation"], Source: labels["teststudio.source"]}, nil
}
