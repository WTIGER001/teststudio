package workerrelay

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type Relay struct {
	listener    net.Listener
	path        string
	mu          sync.Mutex
	connections map[net.Conn]bool
	closed      bool
	wg          sync.WaitGroup
}

func Open(ctx context.Context, path string) (*Relay, error) {
	deadline, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for {
		probe, err := (&net.Dialer{}).DialContext(deadline, "unix", path)
		if err == nil {
			_ = probe.Close()
			break
		}
		select {
		case <-deadline.Done():
			return nil, errors.New("the selected API request gateway is unavailable")
		case <-time.After(100 * time.Millisecond):
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errors.New("the local API request relay is unavailable")
	}
	r := &Relay{listener: listener, path: path, connections: map[net.Conn]bool{}}
	r.wg.Add(1)
	go r.accept(ctx)
	return r, nil
}

func (r *Relay) URL() string { return "http://" + r.listener.Addr().String() }

func (r *Relay) accept(ctx context.Context) {
	defer r.wg.Done()
	for {
		client, err := r.listener.Accept()
		if err != nil {
			return
		}
		r.mu.Lock()
		if r.closed || len(r.connections) >= 128 {
			r.mu.Unlock()
			_ = client.Close()
			continue
		}
		r.connections[client] = true
		r.wg.Add(1)
		r.mu.Unlock()
		go r.pipe(ctx, client)
	}
}

func (r *Relay) pipe(ctx context.Context, client net.Conn) {
	defer r.wg.Done()
	deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
	upstream, err := (&net.Dialer{}).DialContext(deadline, "unix", r.path)
	cancel()
	defer func() { _ = client.Close(); r.mu.Lock(); delete(r.connections, client); r.mu.Unlock() }()
	if err != nil {
		return
	}
	defer func() { _ = upstream.Close(); r.mu.Lock(); delete(r.connections, upstream); r.mu.Unlock() }()
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.connections[upstream] = true
	r.mu.Unlock()
	done := make(chan struct{})
	go func() { _, _ = io.Copy(upstream, client); _ = upstream.Close(); close(done) }()
	_, _ = io.Copy(client, upstream)
	_ = client.Close()
	_ = upstream.Close()
	<-done
}

func (r *Relay) Close() {
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		_ = r.listener.Close()
		for connection := range r.connections {
			_ = connection.Close()
		}
	}
	r.mu.Unlock()
	r.wg.Wait()
}
