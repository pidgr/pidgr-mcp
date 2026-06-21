// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// metadata is the subset of RFC 8414 authorization-server metadata we use.
type metadata struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

const discoveryPath = "/.well-known/oauth-authorization-server"

// discover fetches and parses the authorization-server metadata for the issuer.
func discover(ctx context.Context, hc *http.Client, issuer string) (*metadata, error) {
	url := strings.TrimRight(issuer, "/") + discoveryPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	// G704: the issuer URL is operator-configured (PIDGR_OAUTH_ISSUER), not
	// derived from untrusted request input.
	resp, err := hc.Do(req) //nolint:gosec // G704: issuer is operator-configured, not user-tainted
	if err != nil {
		return nil, fmt.Errorf("fetch discovery document: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("discovery returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var meta metadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decode discovery document: %w", err)
	}

	if meta.AuthorizationEndpoint == "" || meta.TokenEndpoint == "" {
		return nil, fmt.Errorf("discovery document missing authorization_endpoint or token_endpoint")
	}
	return &meta, nil
}
