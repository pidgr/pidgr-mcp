// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package transport

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
)

func TestTokenProviderInterceptorInjectsBearer(t *testing.T) {
	interceptor := tokenProviderInterceptor(func(context.Context) (string, error) {
		return "oauth-access-token", nil
	})

	var captured string
	handler := interceptor(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		captured = req.Header().Get("Authorization")
		return nil, nil
	})

	_, _ = handler(context.Background(), connect.NewRequest(&struct{}{}))

	if captured != "Bearer oauth-access-token" {
		t.Errorf("got Authorization %q, want %q", captured, "Bearer oauth-access-token")
	}
}

func TestTokenProviderInterceptorPropagatesError(t *testing.T) {
	wantErr := errors.New("token unavailable")
	interceptor := tokenProviderInterceptor(func(context.Context) (string, error) {
		return "", wantErr
	})

	called := false
	handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return nil, nil
	})

	_, err := handler(context.Background(), connect.NewRequest(&struct{}{}))
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped token error, got %v", err)
	}
	if called {
		t.Error("downstream handler must not run when token provider fails")
	}
}
