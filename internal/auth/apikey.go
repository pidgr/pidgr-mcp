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

// VerifyAPIKey validates that the token is a well-formed pidgr API key.
// The downstream API performs the actual SHA-256 lookup and RBAC checks —
// this verifier only checks the prefix and minimum length.
func VerifyAPIKey(_ context.Context, token string, _ *http.Request) (*mcpauth.TokenInfo, error) {
	if !isAPIKey(token) {
		return nil, mcpauth.ErrInvalidToken
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
