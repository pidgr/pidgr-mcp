// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

func TestStaticTokenInterceptor(t *testing.T) {
	interceptor := staticTokenInterceptor("pidgr_k_test123")

	var capturedHeader string
	handler := interceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		capturedHeader = req.Header().Get("Authorization")
		return nil, nil
	})

	req := connect.NewRequest(&struct{}{})
	_, _ = handler(context.Background(), req)

	want := "Bearer pidgr_k_test123"
	if capturedHeader != want {
		t.Errorf("got Authorization %q, want %q", capturedHeader, want)
	}
}

func TestDynamicTokenInterceptor(t *testing.T) {
	interceptor := dynamicTokenInterceptor()

	t.Run("with token in context via middleware", func(t *testing.T) {
		// Use the RequireBearerToken middleware to inject TokenInfo into context
		// the same way the real MCP HTTP server does.
		verifier := func(ctx context.Context, token string, r *http.Request) (*mcpauth.TokenInfo, error) {
			return &mcpauth.TokenInfo{
				Scopes:     []string{"openid"},
				Expiration: time.Now().Add(time.Hour),
				Extra:      map[string]any{"raw_token": token},
			}, nil
		}

		var capturedCtx context.Context
		middleware := mcpauth.RequireBearerToken(verifier, nil)
		inner := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedCtx = r.Context()
		}))

		// Simulate an HTTP request with a Bearer token.
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer eyJtest")
		w := httptest.NewRecorder()
		inner.ServeHTTP(w, req)

		if capturedCtx == nil {
			t.Fatal("middleware did not capture context")
		}

		// Now test the interceptor using the captured context.
		var capturedHeader string
		handler := interceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			capturedHeader = req.Header().Get("Authorization")
			return nil, nil
		})

		connectReq := connect.NewRequest(&struct{}{})
		_, _ = handler(capturedCtx, connectReq)

		want := "Bearer eyJtest"
		if capturedHeader != want {
			t.Errorf("got Authorization %q, want %q", capturedHeader, want)
		}
	})

	t.Run("without token in context", func(t *testing.T) {
		var capturedHeader string
		handler := interceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			capturedHeader = req.Header().Get("Authorization")
			return nil, nil
		})

		req := connect.NewRequest(&struct{}{})
		_, _ = handler(context.Background(), req)

		if capturedHeader != "" {
			t.Errorf("expected empty Authorization header, got %q", capturedHeader)
		}
	})
}

func TestNewStaticTokenClients(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	clients := NewStaticTokenClients(ts.URL, "test-key")
	if clients == nil {
		t.Fatal("expected non-nil clients")
		return
	}
	if clients.Campaigns == nil {
		t.Error("expected non-nil Campaigns client")
	}
	if clients.Templates == nil {
		t.Error("expected non-nil Templates client")
	}
	if clients.Groups == nil {
		t.Error("expected non-nil Groups client")
	}
	if clients.Teams == nil {
		t.Error("expected non-nil Teams client")
	}
	if clients.Members == nil {
		t.Error("expected non-nil Members client")
	}
	if clients.Organizations == nil {
		t.Error("expected non-nil Organizations client")
	}
	if clients.Roles == nil {
		t.Error("expected non-nil Roles client")
	}
	if clients.ApiKeys == nil {
		t.Error("expected non-nil ApiKeys client")
	}
	if clients.Heatmaps == nil {
		t.Error("expected non-nil Heatmaps client")
	}
	if clients.Replays == nil {
		t.Error("expected non-nil Replays client")
	}
	if clients.Integrations == nil {
		t.Error("expected non-nil Integrations client")
	}
}

func TestNewDynamicTokenClients(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	clients := NewDynamicTokenClients(ts.URL)
	if clients == nil {
		t.Fatal("expected non-nil clients")
		return
	}
	if clients.Campaigns == nil {
		t.Error("expected non-nil Campaigns client")
	}
	if clients.Integrations == nil {
		t.Error("expected non-nil Integrations client")
	}
}

// TestNewIntegrationsClientUsesIntegrationsBaseURL exercises the Wave 1
// PIDGR_INTEGRATIONS_URL plumbing — when both Static and Dynamic constructors
// accept a distinct base URL for IntegrationsService, they route to it.
func TestNewClientsWithSeparateIntegrationsURL(t *testing.T) {
	apiTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("from", "api")
		w.WriteHeader(http.StatusOK)
	}))
	defer apiTS.Close()
	intTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("from", "integrations")
		w.WriteHeader(http.StatusOK)
	}))
	defer intTS.Close()

	clients := NewStaticTokenClientsWithIntegrationsURL(apiTS.URL, intTS.URL, "key")
	if clients == nil || clients.Integrations == nil {
		t.Fatal("expected non-nil clients + Integrations client")
	}
	if clients.Campaigns == nil {
		t.Fatal("expected non-nil Campaigns client")
	}

	dyn := NewDynamicTokenClientsWithIntegrationsURL(apiTS.URL, intTS.URL)
	if dyn == nil || dyn.Integrations == nil {
		t.Fatal("expected non-nil dynamic clients + Integrations client")
	}
}
