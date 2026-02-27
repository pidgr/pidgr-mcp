// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const testIssuer = "https://auth.example.com/test-pool"

func TestOIDCVerifier_Issuer(t *testing.T) {
	v := NewOIDCVerifier(testIssuer, "")
	if got := v.Issuer(); got != testIssuer {
		t.Errorf("Issuer() = %q, want %q", got, testIssuer)
	}
}

// testKeySetup creates an RSA key pair and JWKS mock server for testing.
type testKeySetup struct {
	jwkKey jwk.Key
	keySet jwk.Set
	server *httptest.Server
}

func newTestKeySetup(t *testing.T) *testKeySetup {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	jwkKey, err := jwk.FromRaw(privateKey)
	if err != nil {
		t.Fatalf("failed to create JWK: %v", err)
	}
	if err := jwkKey.Set(jwk.KeyIDKey, "test-kid"); err != nil {
		t.Fatalf("failed to set kid: %v", err)
	}
	if err := jwkKey.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		t.Fatalf("failed to set alg: %v", err)
	}

	pubKey, err := jwk.FromRaw(privateKey.Public())
	if err != nil {
		t.Fatalf("failed to create public JWK: %v", err)
	}
	if err := pubKey.Set(jwk.KeyIDKey, "test-kid"); err != nil {
		t.Fatalf("failed to set kid: %v", err)
	}
	if err := pubKey.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		t.Fatalf("failed to set alg: %v", err)
	}

	keySet := jwk.NewSet()
	_ = keySet.AddKey(pubKey)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keySet)
	}))

	return &testKeySetup{jwkKey: jwkKey, keySet: keySet, server: ts}
}

func TestOIDCVerifier_ValidToken(t *testing.T) {
	setup := newTestKeySetup(t)
	defer setup.server.Close()

	v := NewOIDCVerifier(testIssuer, "")
	v.jwksURL = setup.server.URL

	token, err := jwt.NewBuilder().
		Issuer(v.issuer).
		Subject("user-123").
		Expiration(time.Now().Add(time.Hour)).
		Claim("custom:org_id", "org-456").
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

	if info.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", info.UserID, "user-123")
	}
	if orgID, ok := info.Extra["org_id"].(string); !ok || orgID != "org-456" {
		t.Errorf("org_id = %v, want %q", info.Extra["org_id"], "org-456")
	}
	if rawToken, ok := info.Extra["raw_token"].(string); !ok || rawToken != string(signed) {
		t.Error("raw_token should be the original token string")
	}
}

func TestOIDCVerifier_ExpiredToken(t *testing.T) {
	setup := newTestKeySetup(t)
	defer setup.server.Close()

	v := NewOIDCVerifier(testIssuer, "")
	v.jwksURL = setup.server.URL

	token, err := jwt.NewBuilder().
		Issuer(v.issuer).
		Subject("user-123").
		Expiration(time.Now().Add(-time.Hour)).
		Build()
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, setup.jwkKey))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = v.Verify(context.Background(), string(signed), nil)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestOIDCVerifier_InvalidSignature(t *testing.T) {
	signingKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifyKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	jwkSignKey, _ := jwk.FromRaw(signingKey)
	_ = jwkSignKey.Set(jwk.KeyIDKey, "signing-kid")
	_ = jwkSignKey.Set(jwk.AlgorithmKey, jwa.RS256)

	pubVerifyKey, _ := jwk.FromRaw(verifyKey.Public())
	_ = pubVerifyKey.Set(jwk.KeyIDKey, "verify-kid")
	_ = pubVerifyKey.Set(jwk.AlgorithmKey, jwa.RS256)

	keySet := jwk.NewSet()
	_ = keySet.AddKey(pubVerifyKey)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keySet)
	}))
	defer ts.Close()

	v := NewOIDCVerifier(testIssuer, "")
	v.jwksURL = ts.URL

	token, _ := jwt.NewBuilder().
		Issuer(v.issuer).
		Subject("user-123").
		Expiration(time.Now().Add(time.Hour)).
		Build()

	signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkSignKey))

	_, err := v.Verify(context.Background(), string(signed), nil)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestOIDCVerifier_JWKSFetchError(t *testing.T) {
	v := NewOIDCVerifier(testIssuer, "")
	v.jwksURL = "http://localhost:1/nonexistent"

	_, err := v.Verify(context.Background(), "some-token", nil)
	if err == nil {
		t.Fatal("expected error when JWKS endpoint is unreachable")
	}
}

func TestOIDCVerifier_GenericErrorMessage(t *testing.T) {
	// All auth errors must return "token validation failed" — never leak details.
	v := NewOIDCVerifier(testIssuer, "")
	v.jwksURL = "http://localhost:1/nonexistent"

	_, err := v.Verify(context.Background(), "bad-token", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("error should contain generic message, got: %q", err.Error())
	}
	if strings.Contains(err.Error(), "localhost") {
		t.Errorf("error should not contain JWKS URL, got: %q", err.Error())
	}
}

func TestOIDCVerifier_IssuerMismatchGenericError(t *testing.T) {
	setup := newTestKeySetup(t)
	defer setup.server.Close()

	v := NewOIDCVerifier(testIssuer, "")
	v.jwksURL = setup.server.URL

	// Token with different issuer.
	token, _ := jwt.NewBuilder().
		Issuer("https://evil.com").
		Subject("user-123").
		Expiration(time.Now().Add(time.Hour)).
		Build()

	signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, setup.jwkKey))

	_, err := v.Verify(context.Background(), string(signed), nil)
	if err == nil {
		t.Fatal("expected error for issuer mismatch")
	}
	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("error should be generic, got: %q", err.Error())
	}
	if strings.Contains(err.Error(), "evil.com") {
		t.Errorf("error should not contain the bad issuer URL, got: %q", err.Error())
	}
}

func TestOIDCVerifier_AudienceValidation(t *testing.T) {
	setup := newTestKeySetup(t)
	defer setup.server.Close()

	t.Run("valid audience", func(t *testing.T) {
		v := NewOIDCVerifier(testIssuer, "my-client-id")
		v.jwksURL = setup.server.URL

		token, _ := jwt.NewBuilder().
			Issuer(v.issuer).
			Subject("user-123").
			Audience([]string{"my-client-id"}).
			Expiration(time.Now().Add(time.Hour)).
			Build()

		signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, setup.jwkKey))

		info, err := v.Verify(context.Background(), string(signed), nil)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if info.UserID != "user-123" {
			t.Errorf("UserID = %q, want %q", info.UserID, "user-123")
		}
	})

	t.Run("wrong audience", func(t *testing.T) {
		v := NewOIDCVerifier(testIssuer, "my-client-id")
		v.jwksURL = setup.server.URL

		token, _ := jwt.NewBuilder().
			Issuer(v.issuer).
			Subject("user-123").
			Audience([]string{"other-client-id"}).
			Expiration(time.Now().Add(time.Hour)).
			Build()

		signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, setup.jwkKey))

		_, err := v.Verify(context.Background(), string(signed), nil)
		if err == nil {
			t.Fatal("expected error for wrong audience")
		}
		if !strings.Contains(err.Error(), "token validation failed") {
			t.Errorf("error should be generic, got: %q", err.Error())
		}
	})

	t.Run("no audience required", func(t *testing.T) {
		v := NewOIDCVerifier(testIssuer, "")
		v.jwksURL = setup.server.URL

		token, _ := jwt.NewBuilder().
			Issuer(v.issuer).
			Subject("user-123").
			Expiration(time.Now().Add(time.Hour)).
			Build()

		signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, setup.jwkKey))

		_, err := v.Verify(context.Background(), string(signed), nil)
		if err != nil {
			t.Fatalf("Verify() should pass without audience check: %v", err)
		}
	})
}

func TestOIDCVerifier_JWKSCacheTTL(t *testing.T) {
	setup := newTestKeySetup(t)
	defer setup.server.Close()

	v := NewOIDCVerifier(testIssuer, "")
	v.jwksURL = setup.server.URL

	// First call fetches JWKS.
	token, _ := jwt.NewBuilder().
		Issuer(v.issuer).
		Subject("user-123").
		Expiration(time.Now().Add(time.Hour)).
		Build()
	signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, setup.jwkKey))

	_, err := v.Verify(context.Background(), string(signed), nil)
	if err != nil {
		t.Fatalf("first Verify() error: %v", err)
	}

	if !v.fetched {
		t.Fatal("expected fetched to be true")
	}
	if v.lastFetched.IsZero() {
		t.Fatal("expected lastFetched to be set")
	}

	// Simulate cache expiry.
	v.mu.Lock()
	v.lastFetched = time.Now().Add(-2 * jwksCacheTTL)
	v.mu.Unlock()

	// getKeySet should trigger a refresh.
	_, err = v.getKeySet(context.Background())
	if err != nil {
		t.Fatalf("getKeySet after TTL expiry error: %v", err)
	}

	v.mu.RLock()
	if time.Since(v.lastFetched) > time.Second {
		t.Error("expected lastFetched to be updated after TTL-based refresh")
	}
	v.mu.RUnlock()
}

func TestNewProtectedResourceMetadata(t *testing.T) {
	issuer := "https://auth.example.com/pool-123"
	metadata := NewProtectedResourceMetadata("https://mcp.pidgr.com", issuer)

	if metadata.Resource != "https://mcp.pidgr.com" {
		t.Errorf("Resource = %q, want %q", metadata.Resource, "https://mcp.pidgr.com")
	}
	if len(metadata.AuthorizationServers) != 1 {
		t.Fatalf("expected 1 authorization server, got %d", len(metadata.AuthorizationServers))
	}
	if metadata.AuthorizationServers[0] != issuer {
		t.Errorf("unexpected authorization server: %s", metadata.AuthorizationServers[0])
	}
	if len(metadata.ScopesSupported) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(metadata.ScopesSupported))
	}
	if len(metadata.BearerMethodsSupported) != 1 || metadata.BearerMethodsSupported[0] != "header" {
		t.Errorf("unexpected bearer methods: %v", metadata.BearerMethodsSupported)
	}
}
