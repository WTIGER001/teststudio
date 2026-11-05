package browsergateway

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

type ownedListener struct {
	net.Listener
	mu          sync.Mutex
	connections map[*ownedConnection]struct{}
	closed      bool
}

type ownedConnection struct {
	net.Conn
	owner *ownedListener
	once  sync.Once
}

func (c *ownedConnection) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() {
		c.owner.mu.Lock()
		delete(c.owner.connections, c)
		c.owner.mu.Unlock()
	})
	return err
}

func (l *ownedListener) Accept() (net.Conn, error) {
	connection, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	owned := &ownedConnection{Conn: connection, owner: l}
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		connection.Close()
		return nil, net.ErrClosed
	}
	l.connections[owned] = struct{}{}
	l.mu.Unlock()
	return owned, nil
}

func (l *ownedListener) Close() error {
	l.mu.Lock()
	l.closed = true
	connections := make([]*ownedConnection, 0, len(l.connections))
	for connection := range l.connections {
		connections = append(connections, connection)
	}
	l.mu.Unlock()
	err := l.Listener.Close()
	for _, connection := range connections {
		connection.Close()
	}
	return err
}

func serveGateway(ctx context.Context, listener net.Listener, handler http.Handler, api bool, settings *tls.Config) error {
	owned := &ownedListener{Listener: listener, connections: map[*ownedConnection]struct{}{}}
	defer owned.Close()
	var incoming net.Listener = owned
	if settings != nil {
		incoming = tls.NewListener(owned, settings)
	}
	server := &http.Server{Handler: handler, BaseContext: func(net.Listener) context.Context { return ctx }, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 64 << 10}
	if api {
		server.ReadTimeout, server.WriteTimeout = 30*time.Second, 30*time.Second
	}
	stopped := context.AfterFunc(ctx, func() { server.Close(); owned.Close() })
	defer stopped()
	err := server.Serve(incoming)
	if errors.Is(err, http.ErrServerClosed) || ctx.Err() != nil && errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func ServeRemote(ctx context.Context, listener net.Listener, configuration Configuration, settings *tls.Config) error {
	gateway, err := NewRemote(configuration)
	if err != nil {
		return err
	}
	defer gateway.Close()
	return gateway.serve(ctx, listener, settings)
}

func (g *Remote) serve(ctx context.Context, listener net.Listener, settings *tls.Config) error {
	if settings == nil || len(settings.Certificates) == 0 {
		return ErrRemoteConfiguration
	}
	settings = settings.Clone()
	if settings.MinVersion < tls.VersionTLS12 {
		settings.MinVersion = tls.VersionTLS12
	}
	settings.NextProtos = []string{"http/1.1"}
	bounded, cancel := context.WithDeadline(ctx, g.session.Deadline)
	defer cancel()
	return serveGateway(bounded, listener, g, g.gateway.requests != nil, settings)
}
