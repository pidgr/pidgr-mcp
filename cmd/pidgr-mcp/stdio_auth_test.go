// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseStdioConfig() *config {
	return &config{
		Transport:       "stdio",
		ApiURL:          "https://api.pidgr.com",
		IntegrationsURL: "https://api.pidgr.com",
		OAuthIssuer:     defaultOAuthIssuer,
		OAuthClientID:   defaultOAuthClientID,
		OAuthScope:      defaultOAuthScope,
	}
}

func TestNewStdioClientsUsesOAuthPath(t *testing.T) {
	cfg := baseStdioConfig()

	// OAuth client construction must succeed without contacting the network;
	// discovery and the browser flow are deferred to the first RPC.
	clients, err := newStdioClients(cfg)
	require.NoError(t, err)
	require.NotNil(t, clients)
	assert.NotNil(t, clients.Campaigns)
}

func TestNewStdioClientsIgnoresPIDGRAPIKey(t *testing.T) {
	// stdio is OAuth-only: PIDGR_API_KEY has no effect, the OAuth path is used.
	t.Setenv("PIDGR_API_KEY", "pidgr_k_abcdefghijklmnop")
	cfg := baseStdioConfig()

	clients, err := newStdioClients(cfg)
	require.NoError(t, err)
	require.NotNil(t, clients)
	assert.NotNil(t, clients.Campaigns)
}

func TestParseConfigStdioRespectsOAuthIssuerOverride(t *testing.T) {
	t.Setenv("PIDGR_MCP_TRANSPORT", "stdio")
	t.Setenv("PIDGR_OAUTH_ISSUER", "https://auth.staging.pidgr.com")

	cfg, err := parseConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://auth.staging.pidgr.com", cfg.OAuthIssuer)
}
