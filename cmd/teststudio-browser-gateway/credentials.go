package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/wtiger001/teststudio/internal/browsergateway"
)

const credentialSocket = "/run/teststudio/credentials.sock"
const maxCredentialBytes = 4 << 20

func receiveCredential(ctx context.Context, path string) (*browsergateway.Configuration, error) {
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, errors.New("API credential initialization is unavailable")
	}
	defer listener.Close()
	defer os.Remove(path)
	if os.Chmod(path, 0600) != nil {
		return nil, errors.New("API credential initialization is unavailable")
	}
	stop := context.AfterFunc(ctx, func() { _ = listener.Close() })
	defer stop()
	connection, err := listener.Accept()
	if err != nil {
		return nil, errors.New("API credential initialization interrupted")
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
	data, err := io.ReadAll(io.LimitReader(connection, maxCredentialBytes+1))
	if err != nil || len(data) > maxCredentialBytes {
		return nil, errors.New("API credential initialization exceeds its input limit")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var value browsergateway.Configuration
	if decoder.Decode(&value) != nil {
		return nil, errors.New("API credential initialization is invalid")
	}
	var extra any
	if !errors.Is(decoder.Decode(&extra), io.EOF) {
		return nil, errors.New("API credential initialization has trailing data")
	}
	return &value, nil
}

func sendCredential(ctx context.Context, path string, input io.Reader) error {
	data, err := io.ReadAll(io.LimitReader(input, maxCredentialBytes+1))
	if err != nil || len(data) > maxCredentialBytes {
		return errors.New("API credential initialization exceeds its input limit")
	}
	for ctx.Err() == nil {
		connection, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
		if err == nil {
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
			_, err = connection.Write(data)
			if err != nil {
				return errors.New("API credential initialization transfer failed")
			}
			return nil
		}
		select {
		case <-ctx.Done():
		case <-time.After(50 * time.Millisecond):
		}
	}
	return errors.New("API credential initialization is unavailable")
}
