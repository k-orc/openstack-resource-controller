package main

import (
	"context"
	"time"
)

var registeredLimitIds []string

func onStart(ctx context.Context) (retErr error) {
	c, err := newKeystoneClient(ctx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if err := setupRegisteredLimits(ctx, c); err != nil {
		return err
	}

	return nil
}

func onStop(ctx context.Context) error {
	c, err := newKeystoneClient(ctx)
	if err != nil {
		return err
	}

	cleanUpRegisteredLimits(ctx, c)
	cleanUpDomains(ctx, c)

	return nil
}

func init() {
	onStartFunc = onStart
	onStopFunc = onStop
}
