// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientValidation(t *testing.T) {
	_, err := NewClient(Config{ClientID: "pidgr-mcp"})
	require.Error(t, err)

	_, err = NewClient(Config{Issuer: "https://auth.pidgr.com"})
	require.Error(t, err)
}

func TestNewClientAppliesDefaults(t *testing.T) {
	c, err := NewClient(Config{Issuer: "https://auth.pidgr.com", ClientID: "pidgr-mcp"})
	require.NoError(t, err)
	assert.NotNil(t, c.cfg.Store)
	assert.NotNil(t, c.cfg.Opener)
	assert.NotNil(t, c.cfg.HTTPClient)
	assert.NotNil(t, c.now)
}

func TestBuildAuthorizeURL(t *testing.T) {
	raw, err := buildAuthorizeURL("https://auth.pidgr.com/authorize", authorizeParams{
		clientID:    "pidgr-mcp",
		redirectURI: "http://127.0.0.1:54321/callback",
		scope:       "openid offline_access",
		state:       "st-1",
		challenge:   "chal-1",
		method:      "S256",
	})
	require.NoError(t, err)

	u, err := url.Parse(raw)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "pidgr-mcp", q.Get("client_id"))
	assert.Equal(t, "http://127.0.0.1:54321/callback", q.Get("redirect_uri"))
	assert.Equal(t, "chal-1", q.Get("code_challenge"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
	assert.Equal(t, "st-1", q.Get("state"))
	assert.Equal(t, "openid offline_access", q.Get("scope"))
}

func TestBuildAuthorizeURLOmitsEmptyScope(t *testing.T) {
	raw, err := buildAuthorizeURL("https://auth.pidgr.com/authorize", authorizeParams{
		clientID:    "pidgr-mcp",
		redirectURI: "http://127.0.0.1:1/callback",
		state:       "st",
		challenge:   "c",
		method:      "S256",
	})
	require.NoError(t, err)
	u, _ := url.Parse(raw)
	_, hasScope := u.Query()["scope"]
	assert.False(t, hasScope)
}
