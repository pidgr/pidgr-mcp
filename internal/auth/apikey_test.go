// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
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

func TestVerifyAPIKey_Valid(t *testing.T) {
	apiKey := "pidgr_k_test1234567890ab" //nolint:gosec // G101: test fixture, not a credential
	info, err := VerifyAPIKey(context.Background(), apiKey, nil)
	if err != nil {
		t.Fatalf("VerifyAPIKey() error: %v", err)
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

func TestVerifyAPIKey_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"jwt token", "eyJhbGciOiJSUzI1NiJ9.payload.sig"},
		{"wrong prefix", "sk_live_abc1234567890123"},
		{"too short", "pidgr_k_short"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyAPIKey(context.Background(), tt.token, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, mcpauth.ErrInvalidToken) {
				t.Errorf("error should be ErrInvalidToken, got: %v", err)
			}
		})
	}
}
