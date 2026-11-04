package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

func serveServers(ctx context.Context, servers []*http.Server, ready func() func()) error {
	listeners := []net.Listener{}
	var cleanup func()
	defer func() {
		for _, listener := range listeners {
			listener.Close()
		}
		for _, server := range servers {
			server.Close()
		}
		if cleanup != nil {
			cleanup()
		}
	}()
	for _, server := range servers {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			return err
		}
		listeners = append(listeners, listener)
	}
	if ready != nil {
		cleanup = ready()
	}
	failed := make(chan error, len(servers))
	for i, server := range servers {
		go func(server *http.Server, listener net.Listener) {
			if server.TLSConfig != nil {
				failed <- server.ServeTLS(listener, "", "")
				return
			}
			failed <- server.Serve(listener)
		}(server, listeners[i])
	}
	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		failures := make(chan error, len(servers))
		for _, server := range servers {
			go func(server *http.Server) { failures <- server.Shutdown(shutdown) }(server)
		}
		var result error
		for range servers {
			result = errors.Join(result, <-failures)
		}
		return result
	}
	return nil
}
