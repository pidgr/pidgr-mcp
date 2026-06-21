// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverMetadata(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"issuer": "https://auth.pidgr.com",
			"authorization_endpoint": "https://auth.pidgr.com/authorize",
			"token_endpoint": "https://auth.pidgr.com/token",
			"code_challenge_methods_supported": ["S256"]
		}`))
	}))
	defer srv.Close()

	meta, err := discover(context.Background(), srv.Client(), srv.URL)
	require.NoError(t, err)

	assert.Equal(t, "/.well-known/oauth-authorization-server", gotPath)
	assert.Equal(t, "https://auth.pidgr.com", meta.Issuer)
	assert.Equal(t, "https://auth.pidgr.com/authorize", meta.AuthorizationEndpoint)
	assert.Equal(t, "https://auth.pidgr.com/token", meta.TokenEndpoint)
}

func TestDiscoverMetadataTrimsTrailingSlash(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"issuer":"x","authorization_endpoint":"a","token_endpoint":"t"}`))
	}))
	defer srv.Close()

	_, err := discover(context.Background(), srv.Client(), srv.URL+"/")
	require.NoError(t, err)
	assert.Equal(t, "/.well-known/oauth-authorization-server", gotPath)
}

func TestDiscoverMetadataMissingEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"issuer":"https://auth.pidgr.com"}`))
	}))
	defer srv.Close()

	_, err := discover(context.Background(), srv.Client(), srv.URL)
	require.Error(t, err)
}

func TestDiscoverMetadataHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := discover(context.Background(), srv.Client(), srv.URL)
	require.Error(t, err)
}
