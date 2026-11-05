package main

import (
	"context"
	"errors"
	"time"

	"github.com/wtiger001/teststudio/adapters/registry/acr"
	"github.com/wtiger001/teststudio/internal/hostedexecution"
)

func awaitAzureRegistry(ctx context.Context, open func(context.Context) (*acr.Client, error), interval time.Duration) (*acr.Client, error) {
	for {
		if ctx.Err() != nil {
			return nil, hostedexecution.ErrUnavailable
		}
		client, err := open(ctx)
		if err == nil {
			return client, nil
		}
		if !errors.Is(err, hostedexecution.ErrUnavailable) {
			return nil, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, hostedexecution.ErrUnavailable
		case <-timer.C:
		}
	}
}
