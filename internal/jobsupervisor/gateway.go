package jobsupervisor

import (
	"context"

	"github.com/wtiger001/teststudio/internal/browsergateway"
	"github.com/wtiger001/teststudio/internal/execution"
	"github.com/wtiger001/teststudio/internal/workerprotocol"
)

type gatewayTransport interface {
	GatewayConnection(context.Context) (browsergateway.RemoteConnection, error)
}

func (s *Supervisor) startGateway(ctx context.Context, packet workerprotocol.Packet, input execution.CodedInput) (func() error, error) {
	transport, ok := s.Transport.(gatewayTransport)
	if !ok || s.GatewaySocket == "" {
		return nil, ErrWork
	}
	var connection browsergateway.RemoteConnection
	if err := s.retry(ctx, func() error { var err error; connection, err = transport.GatewayConnection(ctx); return err }); err != nil {
		return nil, ErrWork
	}
	if connection.Session.Execution != packet.Execution || !connection.Session.Deadline.Equal(packet.Deadline) {
		return nil, ErrWork
	}
	return browsergateway.StartRemoteBridge(ctx, s.GatewaySocket, *input.Browser, connection)
}
