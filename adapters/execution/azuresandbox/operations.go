package azuresandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func (c *Client) ImportImage(ctx context.Context, plan *ImagePlan) (DiskImage, error) {
	if plan == nil || hash(plan.config) != hash(c.config) || plan.valid() != nil {
		return DiskImage{}, hostedexecution.ErrConfiguration
	}
	body := plan.definition()
	body["managedIdentityResourceId"] = plan.proof.PullIdentity
	data, err := c.request(ctx, http.MethodPut, c.target("/diskimages"), body)
	if err != nil {
		return DiskImage{}, err
	}
	image, err := c.image(data)
	if err != nil {
		return DiskImage{}, err
	}
	if plan.Verify(image) != nil {
		return DiskImage{}, hostedexecution.ErrOwnership
	}
	return image, nil
}

func (c *Client) ReadImage(ctx context.Context, id string) (DiskImage, error) {
	if !identifier.MatchString(id) {
		return DiskImage{}, hostedexecution.ErrOwnership
	}
	data, err := c.request(ctx, http.MethodGet, c.target("/diskimages/"+id), nil)
	if err != nil {
		return DiskImage{}, err
	}
	image, err := c.image(data)
	if err != nil {
		return DiskImage{}, err
	}
	if image.ID != id {
		return DiskImage{}, hostedexecution.ErrOwnership
	}
	return image, nil
}

func (c *Client) image(data []byte) (DiskImage, error) {
	var value imageResponse
	if json.Unmarshal(data, &value) != nil {
		return DiskImage{}, hostedexecution.ErrUnavailable
	}
	if !identifier.MatchString(value.ID) || !c.config.validImage(value.Image.Base) || !slices.Equal(value.Image.Entrypoint, []string{"/runner/teststudio-job"}) || len(value.Image.Command) != 0 {
		return DiskImage{}, hostedexecution.ErrOwnership
	}
	if !identifier.MatchString(value.Status.State) {
		return DiskImage{}, hostedexecution.ErrUnavailable
	}
	definition := map[string]any{"image": map[string]any{"base": value.Image.Base, "entrypoint": value.Image.Entrypoint, "cmd": []string{}}, "labels": value.Labels, "dockerfileContent": value.Dockerfile}
	return DiskImage{ID: value.ID, Base: value.Image.Base, State: value.Status.State, Entrypoint: value.Image.Entrypoint, Command: value.Image.Command, DefinitionHash: hash(definition)}, nil
}

func (c *Client) Create(ctx context.Context, plan *Plan) (Snapshot, error) {
	if plan == nil || hash(plan.config) != hash(c.config) {
		return Snapshot{}, hostedexecution.ErrOwnership
	}
	body, err := plan.launch(time.Now())
	if err != nil {
		return Snapshot{}, err
	}
	ctx, cancel := context.WithDeadline(ctx, plan.proof.Deadline)
	defer cancel()
	data, err := c.request(ctx, http.MethodPut, c.target("/sandboxes"), body)
	if err != nil {
		return Snapshot{}, err
	}
	actual, err := snapshot(data)
	if err != nil {
		return Snapshot{}, err
	}
	if err := plan.Verify(actual); err != nil {
		return Snapshot{}, err
	}
	return actual, nil
}

func (c *Client) Read(ctx context.Context, id string) (Snapshot, error) {
	if !identifier.MatchString(id) {
		return Snapshot{}, hostedexecution.ErrOwnership
	}
	data, err := c.request(ctx, http.MethodGet, c.target("/sandboxes/"+id), nil)
	if err != nil {
		return Snapshot{}, err
	}
	actual, err := snapshot(data)
	if err != nil {
		return Snapshot{}, err
	}
	if actual.ID != id {
		return Snapshot{}, hostedexecution.ErrOwnership
	}
	return actual, nil
}

func (c *Client) Stop(ctx context.Context, id string) error {
	if !identifier.MatchString(id) {
		return hostedexecution.ErrOwnership
	}
	_, err := c.request(ctx, http.MethodPost, c.target("/sandboxes/"+id+"/stop"), nil)
	return err
}

func (c *Client) Delete(ctx context.Context, id string) error {
	if !identifier.MatchString(id) {
		return hostedexecution.ErrOwnership
	}
	_, err := c.request(ctx, http.MethodDelete, c.target("/sandboxes/"+id), nil)
	return err
}
