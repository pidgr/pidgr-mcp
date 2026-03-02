// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

const (
	apiKeyPrefix = "pidgr_k_" //nolint:gosec // G101: prefix pattern, not a credential
	// Minimum total length: prefix (8) + at least 16 chars of key material.
	apiKeyMinLen = 24
	// Synthetic expiration for API key tokens. The actual expiry is enforced
	// by the API's database lookup — this value only satisfies the MCP SDK's
	// RequireBearerToken middleware which rejects zero/past expirations.
	apiKeyTTL = 24 * time.Hour
)

// CompositeVerifier delegates token verification to either an API key
// pass-through path or an OIDC JWT verifier based on the token prefix.
type CompositeVerifier struct {
	oidc *OIDCVerifier
}

// NewCompositeVerifier wraps an OIDCVerifier with API key detection.
func NewCompositeVerifier(oidc *OIDCVerifier) *CompositeVerifier {
	return &CompositeVerifier{oidc: oidc}
}

// Verify implements auth.TokenVerifier for the MCP SDK.
// Tokens with the pidgr_k_ prefix are passed through without cryptographic
// validation — the downstream API performs SHA-256 lookup and RBAC checks.
// All other tokens are delegated to the OIDC verifier.
func (v *CompositeVerifier) Verify(ctx context.Context, token string, req *http.Request) (*mcpauth.TokenInfo, error) {
	if !isAPIKey(token) {
		return v.oidc.Verify(ctx, token, req)
	}

	return &mcpauth.TokenInfo{
		Expiration: time.Now().Add(apiKeyTTL),
		Extra: map[string]any{
			"raw_token": token,
		},
	}, nil
}

// isAPIKey reports whether the token looks like a pidgr API key.
func isAPIKey(token string) bool {
	return len(token) >= apiKeyMinLen && strings.HasPrefix(token, apiKeyPrefix)
}
