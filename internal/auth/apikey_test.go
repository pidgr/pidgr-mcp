// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestIsAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"valid key", "pidgr_k_abc1234567890123", true},
		{"valid long key", "pidgr_k_abcdefghijklmnopqrstuvwxyz0123456789", true},
		{"exactly min length", "pidgr_k_1234567890123456", true},
		{"prefix only", "pidgr_k_", false},
		{"too short", "pidgr_k_short", false},
		{"jwt token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload.sig", false},
		{"empty string", "", false},
		{"wrong prefix", "sk_live_abc1234567890123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAPIKey(tt.token); got != tt.want {
				t.Errorf("isAPIKey(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}

func TestCompositeVerifier_APIKeyPassthrough(t *testing.T) {
	// OIDC verifier is not used for API keys — pass nil-like verifier.
	oidc := NewOIDCVerifier(testIssuer, "")
	v := NewCompositeVerifier(oidc)

	apiKey := "pidgr_k_test1234567890ab"
	info, err := v.Verify(context.Background(), apiKey, nil)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}

	// raw_token must be set for dynamicTokenInterceptor.
	rawToken, ok := info.Extra["raw_token"].(string)
	if !ok || rawToken != apiKey {
		t.Errorf("raw_token = %v, want %q", info.Extra["raw_token"], apiKey)
	}

	// Expiration must be in the future (SDK rejects zero/past).
	if info.Expiration.Before(time.Now()) {
		t.Error("Expiration should be in the future")
	}
	if info.Expiration.After(time.Now().Add(25 * time.Hour)) {
		t.Error("Expiration should be roughly 24h from now")
	}

	// UserID must be empty — API key identity is resolved by the backend.
	if info.UserID != "" {
		t.Errorf("UserID = %q, want empty", info.UserID)
	}
}

func TestCompositeVerifier_JWTDelegation(t *testing.T) {
	setup := newTestKeySetup(t)
	defer setup.server.Close()

	oidc := NewOIDCVerifier(testIssuer, "")
	oidc.jwksURL = setup.server.URL
	v := NewCompositeVerifier(oidc)

	token, err := jwt.NewBuilder().
		Issuer(testIssuer).
		Subject("user-789").
		Expiration(time.Now().Add(time.Hour)).
		Claim("custom:org_id", "org-012").
		Build()
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, setup.jwkKey))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	info, err := v.Verify(context.Background(), string(signed), nil)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}

	// OIDC path sets UserID from sub claim.
	if info.UserID != "user-789" {
		t.Errorf("UserID = %q, want %q", info.UserID, "user-789")
	}
	if orgID, ok := info.Extra["org_id"].(string); !ok || orgID != "org-012" {
		t.Errorf("org_id = %v, want %q", info.Extra["org_id"], "org-012")
	}
}

func TestCompositeVerifier_InvalidJWT(t *testing.T) {
	// Non-API-key token that fails OIDC validation should return an error.
	oidc := NewOIDCVerifier(testIssuer, "")
	oidc.jwksURL = "http://localhost:1/nonexistent"
	v := NewCompositeVerifier(oidc)

	_, err := v.Verify(context.Background(), "not-an-api-key", nil)
	if err == nil {
		t.Fatal("expected error for invalid JWT, got nil")
	}
}
