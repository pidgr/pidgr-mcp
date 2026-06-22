// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAuthMiddleware mimics the real bearer-token middleware: returns 401
// when no Authorization header is present, otherwise passes through.
func stubAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// okHandler is a stand-in for the MCP streamable HTTP handler.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// prmHandler is a stand-in for the protected-resource metadata handler.
var prmHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(oauthex.ProtectedResourceMetadata{
		Resource:             "https://mcp.pidgr.com",
		AuthorizationServers: []string{"https://auth.pidgr.com"},
	})
})

func TestHealthzReturns200WithoutAuth(t *testing.T) {
	mux := newHTTPMux(stubAuthMiddleware, okHandler, prmHandler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestHealthzDoesNotRequireBearerToken(t *testing.T) {
	mux := newHTTPMux(stubAuthMiddleware, okHandler, prmHandler)

	// No Authorization header
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestProtectedResourceMetadataServed asserts the resource server advertises the
// Pidgr OAuth provider as its authorization server (RFC 9728), unauthenticated,
// so MCP clients can discover where to authenticate.
func TestProtectedResourceMetadataServed(t *testing.T) {
	mux := newHTTPMux(stubAuthMiddleware, okHandler, prmHandler)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var prm oauthex.ProtectedResourceMetadata
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prm))
	assert.Contains(t, prm.AuthorizationServers, "https://auth.pidgr.com")
}

func TestRootRequiresAuth(t *testing.T) {
	mux := newHTTPMux(stubAuthMiddleware, okHandler, prmHandler)

	// Without Authorization header → 401
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// With Authorization header → passes through (stub middleware; real OAuth
	// verification is covered by oauth.Verifier tests).
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer some-oauth-jwt")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestProtectedResourceMetadataFromConfig(t *testing.T) {
	cfg := &config{
		OAuthIssuer: "https://auth.pidgr.com",
		ResourceURL: "https://mcp.pidgr.com",
		OAuthScope:  "campaigns:read campaigns:write",
	}
	prm := protectedResourceMetadata(cfg)

	assert.Equal(t, "https://mcp.pidgr.com", prm.Resource)
	assert.Equal(t, []string{"https://auth.pidgr.com"}, prm.AuthorizationServers)
	assert.Contains(t, prm.BearerMethodsSupported, "header")
	assert.ElementsMatch(t, []string{"campaigns:read", "campaigns:write"}, prm.ScopesSupported)
}
