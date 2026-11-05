package client

import (
	"context"
	"net/http"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

func (c *Client) GatewayInput(ctx context.Context) (browsergateway.Configuration, error) {
	var configuration browsergateway.Configuration
	err := c.request(ctx, http.MethodGet, "/gateway-input", nil, &configuration, 4<<20)
	if err == nil && (configuration.Remote == nil || configuration.Remote.Execution != c.execution) {
		err = workerprotocol.ErrInvalid
	}
	return configuration, err
}

func (c *Client) GatewayConnection(ctx context.Context) (browsergateway.RemoteConnection, error) {
	var connection browsergateway.RemoteConnection
	err := c.request(ctx, http.MethodGet, "/gateway-connection", nil, &connection, 128<<10)
	if err == nil && connection.Session.Execution != c.execution {
		err = workerprotocol.ErrInvalid
	}
	return connection, err
}
