package oci

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/domain"
	"github.com/wtiger001/teststudio/internal/execution"
)

type Runtime struct {
	Runner, PlaywrightVersion   string
	GatewayTransport, APIPolicy string
	SupervisorRevision          string
	GatewayAddress              string
	template                    *runtimeTemplate
}

type runtimeTemplate struct {
	client        *Client
	repository    string
	manifest      manifest
	configuration []byte
	labels        map[string]string
}

type SourceLayer struct {
	Reference    execution.Reference
	SourceDigest string
	DiffID       string
	Compressed   []byte
}

type UploadJournal interface {
	Opening(string) error
	Opened(string) error
	Completed() error
}

var ErrPublication = errors.New("source registry publication could not be confirmed")

func (c *Client) SourceRuntime(ctx context.Context, reference string) (Runtime, error) {
	metadata, err := c.inspect(ctx, reference, false)
	if err != nil {
		return Runtime{}, err
	}
	if metadata.Image.Discovery != "worker-exchange/v1" {
		return Runtime{}, ErrInvalid
	}
	for _, key := range []string{"teststudio.project", "teststudio.installation", "teststudio.preparation", "teststudio.source"} {
		if metadata.template.labels[key] != "" {
			return Runtime{}, ErrInvalid
		}
	}
	return Runtime{GatewayAddress: metadata.Image.GatewayAddress, SupervisorRevision: metadata.Image.SupervisorRevision, Runner: metadata.Image.Runner, PlaywrightVersion: metadata.template.labels["teststudio.playwright-version"], GatewayTransport: metadata.Image.GatewayTransport, APIPolicy: metadata.Image.APIPolicy, template: metadata.template}, nil
}

func (c *Client) PublishSource(ctx context.Context, runtime Runtime, layer SourceLayer, journal UploadJournal) (Metadata, error) {
	ref, base := layer.Reference, runtime.template
	if base == nil || base.client != c || journal == nil || !domain.ValidID(ref.ID) || !domain.ValidID(ref.Project) || !domain.ValidID(ref.Installation) || !strings.HasPrefix(ref.Installation, "preparation-") || !digest.MatchString(layer.SourceDigest) || !digest.MatchString(layer.DiffID) || len(layer.Compressed) < 1 || len(layer.Compressed) > 144<<20 || len(base.manifest.Layers) >= 128 {
		return Metadata{}, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	configuration, image, err := sourceConfiguration(base, layer)
	if err != nil {
		return Metadata{}, err
	}
	credential, err := credential(ctx, c.credentials)
	if err != nil {
		return Metadata{}, err
	}
	auth := &authorization{base: credential, current: credential}
	for _, blob := range [][]byte{layer.Compressed, configuration} {
		if err := c.pushBlob(ctx, base.repository, blob, auth, journal); err != nil {
			return Metadata{}, err
		}
	}
	body, err := json.Marshal(image)
	if err != nil || len(body) > metadataLimit {
		return Metadata{}, ErrInvalid
	}
	hash := contentDigest(body)
	response, err := c.writeRequest(ctx, base.repository, "PUT", c.origin+"/v2/"+base.repository+"/manifests/"+hash, body, ociManifest, auth)
	if err == nil {
		response.Body.Close()
		if response.StatusCode != 201 || !matchingDigest(response, hash) {
			err = ErrPublication
		}
	}
	reference := strings.TrimPrefix(c.origin, "https://") + "/" + base.repository + "@" + hash
	verification, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer stop()
	result, inspectErr := c.Inspect(verification, reference)
	if inspectErr != nil || result.Project != ref.Project || result.Installation != ref.Installation || result.Preparation != ref.ID || result.Source != layer.SourceDigest || result.Configuration != image.Config.Digest {
		return Metadata{}, errors.Join(ErrPublication, err)
	}
	return result, nil
}

func sourceConfiguration(base *runtimeTemplate, layer SourceLayer) ([]byte, manifest, error) {
	var document map[string]json.RawMessage
	var config map[string]json.RawMessage
	var filesystem struct {
		Type    string   `json:"type"`
		DiffIDs []string `json:"diff_ids"`
	}
	if json.Unmarshal(base.configuration, &document) != nil || json.Unmarshal(document["config"], &config) != nil || json.Unmarshal(document["rootfs"], &filesystem) != nil {
		return nil, manifest{}, ErrInvalid
	}
	labels := map[string]string{}
	for key, value := range base.labels {
		labels[key] = value
	}
	labels["teststudio.project"] = layer.Reference.Project
	labels["teststudio.installation"] = layer.Reference.Installation
	labels["teststudio.preparation"] = layer.Reference.ID
	labels["teststudio.source"] = layer.SourceDigest
	config["Labels"], _ = json.Marshal(labels)
	document["config"], _ = json.Marshal(config)
	filesystem.DiffIDs = append(filesystem.DiffIDs, layer.DiffID)
	document["rootfs"], _ = json.Marshal(filesystem)
	var history []json.RawMessage
	if value := document["history"]; len(value) > 0 && json.Unmarshal(value, &history) != nil {
		return nil, manifest{}, ErrInvalid
	}
	history = append(history, json.RawMessage(`{"created_by":"TestStudio source assembly"}`))
	document["history"], _ = json.Marshal(history)
	data, err := json.Marshal(document)
	if err != nil || len(data) > metadataLimit {
		return nil, manifest{}, ErrInvalid
	}
	image := manifest{Schema: 2, MediaType: ociManifest, Config: descriptor{MediaType: "application/vnd.oci.image.config.v1+json", Digest: contentDigest(data), Size: int64(len(data))}, Layers: append(append([]descriptor{}, base.manifest.Layers...), descriptor{MediaType: "application/vnd.oci.image.layer.v1.tar+gzip", Digest: contentDigest(layer.Compressed), Size: int64(len(layer.Compressed))})}
	return data, image, nil
}
