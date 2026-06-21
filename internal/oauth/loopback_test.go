// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoopbackCapturesCode(t *testing.T) {
	lb, err := newLoopback("expected-state")
	require.NoError(t, err)
	defer lb.Close()

	assert.Contains(t, lb.RedirectURI(), "http://127.0.0.1:")
	assert.Contains(t, lb.RedirectURI(), "/callback")

	go func() {
		resp, derr := http.Get(lb.RedirectURI() + "?code=the-code&state=expected-state")
		if derr == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	code, err := lb.Wait(ctx)
	require.NoError(t, err)
	assert.Equal(t, "the-code", code)
}

func TestLoopbackRejectsStateMismatch(t *testing.T) {
	lb, err := newLoopback("expected-state")
	require.NoError(t, err)
	defer lb.Close()

	go func() {
		resp, derr := http.Get(lb.RedirectURI() + "?code=the-code&state=WRONG")
		if derr == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = lb.Wait(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state")
}

func TestLoopbackPropagatesProviderError(t *testing.T) {
	lb, err := newLoopback("expected-state")
	require.NoError(t, err)
	defer lb.Close()

	go func() {
		resp, derr := http.Get(lb.RedirectURI() + "?error=access_denied&error_description=user+said+no&state=expected-state")
		if derr == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = lb.Wait(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_denied")
}

func TestLoopbackWaitRespectsContext(t *testing.T) {
	lb, err := newLoopback("expected-state")
	require.NoError(t, err)
	defer lb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = lb.Wait(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
