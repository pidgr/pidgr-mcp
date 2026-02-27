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
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestCognitoVerifier_Issuer(t *testing.T) {
	v := NewCognitoVerifier("us-east-1_abc123", "us-east-1")
	want := "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_abc123"
	if got := v.Issuer(); got != want {
		t.Errorf("Issuer() = %q, want %q", got, want)
	}
}

func TestCognitoVerifier_ValidToken(t *testing.T) {
	// Generate RSA key pair.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Create JWK key.
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

	// Create public key set for JWKS endpoint.
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
	keySet.AddKey(pubKey)

	// Start mock JWKS server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keySet)
	}))
	defer ts.Close()

	// Create verifier pointing to mock server.
	poolID := "us-east-1_test"
	region := "us-east-1"
	v := NewCognitoVerifier(poolID, region)
	v.jwksURL = ts.URL // Override JWKS URL.

	// Build a valid JWT.
	token, err := jwt.NewBuilder().
		Issuer(v.issuer).
		Subject("user-123").
		Expiration(time.Now().Add(time.Hour)).
		Claim("custom:org_id", "org-456").
		Build()
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkKey))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Verify the token.
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

func TestCognitoVerifier_ExpiredToken(t *testing.T) {
	// Generate RSA key pair.
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
	keySet.AddKey(pubKey)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keySet)
	}))
	defer ts.Close()

	v := NewCognitoVerifier("us-east-1_test", "us-east-1")
	v.jwksURL = ts.URL

	// Build an expired JWT.
	token, err := jwt.NewBuilder().
		Issuer(v.issuer).
		Subject("user-123").
		Expiration(time.Now().Add(-time.Hour)).
		Build()
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkKey))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = v.Verify(context.Background(), string(signed), nil)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestCognitoVerifier_InvalidSignature(t *testing.T) {
	// Generate two different key pairs.
	signingKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	verifyKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	jwkSignKey, _ := jwk.FromRaw(signingKey)
	jwkSignKey.Set(jwk.KeyIDKey, "signing-kid")
	jwkSignKey.Set(jwk.AlgorithmKey, jwa.RS256)

	pubVerifyKey, _ := jwk.FromRaw(verifyKey.Public())
	pubVerifyKey.Set(jwk.KeyIDKey, "verify-kid")
	pubVerifyKey.Set(jwk.AlgorithmKey, jwa.RS256)

	keySet := jwk.NewSet()
	keySet.AddKey(pubVerifyKey)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keySet)
	}))
	defer ts.Close()

	v := NewCognitoVerifier("us-east-1_test", "us-east-1")
	v.jwksURL = ts.URL

	token, _ := jwt.NewBuilder().
		Issuer(v.issuer).
		Subject("user-123").
		Expiration(time.Now().Add(time.Hour)).
		Build()

	// Sign with the wrong key.
	signed, _ := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkSignKey))

	_, err := v.Verify(context.Background(), string(signed), nil)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestCognitoVerifier_JWKSFetchError(t *testing.T) {
	v := NewCognitoVerifier("us-east-1_test", "us-east-1")
	v.jwksURL = "http://localhost:1/nonexistent"

	_, err := v.Verify(context.Background(), "some-token", nil)
	if err == nil {
		t.Fatal("expected error when JWKS endpoint is unreachable")
	}
}

func TestNewProtectedResourceMetadata(t *testing.T) {
	metadata := NewProtectedResourceMetadata("https://mcp.pidgr.com", "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_test")

	if metadata.Resource != "https://mcp.pidgr.com" {
		t.Errorf("Resource = %q, want %q", metadata.Resource, "https://mcp.pidgr.com")
	}
	if len(metadata.AuthorizationServers) != 1 {
		t.Fatalf("expected 1 authorization server, got %d", len(metadata.AuthorizationServers))
	}
	if metadata.AuthorizationServers[0] != "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_test" {
		t.Errorf("unexpected authorization server: %s", metadata.AuthorizationServers[0])
	}
	if len(metadata.ScopesSupported) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(metadata.ScopesSupported))
	}
	if len(metadata.BearerMethodsSupported) != 1 || metadata.BearerMethodsSupported[0] != "header" {
		t.Errorf("unexpected bearer methods: %v", metadata.BearerMethodsSupported)
	}
}
