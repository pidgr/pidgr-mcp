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

func TestNewStdioClientsWithAPIKeyUsesStaticPath(t *testing.T) {
	cfg := baseStdioConfig()
	cfg.apiKey = "pidgr_k_abcdefghijklmnop"

	clients, err := newStdioClients(cfg)
	require.NoError(t, err)
	require.NotNil(t, clients)
	assert.NotNil(t, clients.Campaigns)
}

func TestNewStdioClientsWithoutAPIKeyUsesOAuthPath(t *testing.T) {
	cfg := baseStdioConfig()
	cfg.apiKey = ""

	// OAuth client construction must succeed without contacting the network;
	// discovery and the browser flow are deferred to the first RPC.
	clients, err := newStdioClients(cfg)
	require.NoError(t, err)
	require.NotNil(t, clients)
	assert.NotNil(t, clients.Campaigns)
}

func TestParseConfigStdioAllowsMissingAPIKey(t *testing.T) {
	t.Setenv("PIDGR_MCP_TRANSPORT", "stdio")
	t.Setenv("PIDGR_API_KEY", "")

	cfg, err := parseConfig()
	require.NoError(t, err)
	assert.Equal(t, "stdio", cfg.Transport)
	assert.Empty(t, cfg.apiKey)
	assert.Equal(t, defaultOAuthClientID, cfg.OAuthClientID)
}

func TestParseConfigStdioRespectsOAuthIssuerOverride(t *testing.T) {
	t.Setenv("PIDGR_MCP_TRANSPORT", "stdio")
	t.Setenv("PIDGR_API_KEY", "")
	t.Setenv("PIDGR_OAUTH_ISSUER", "https://auth.staging.pidgr.com")

	cfg, err := parseConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://auth.staging.pidgr.com", cfg.OAuthIssuer)
}
