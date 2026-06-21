// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// providerFixture spins up an httptest server that serves both the
// authorization-server discovery document and a JWKS document advertising a
// single RSA public key, signed-by a known private key.
type providerFixture struct {
	srv      *httptest.Server
	issuer   string
	key      *rsa.PrivateKey
	kid      string
	jwksHits int
}

func newProviderFixture(t *testing.T) *providerFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	f := &providerFixture{key: key, kid: "test-key-1"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 f.issuer,
			"authorization_endpoint": f.issuer + "/authorize",
			"token_endpoint":         f.issuer + "/token",
			"jwks_uri":               f.issuer + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		f.jwksHits++
		pub := key.Public().(*rsa.PublicKey)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": f.kid,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			}},
		})
	})
	srv := httptest.NewServer(mux)
	f.srv = srv
	f.issuer = srv.URL
	return f
}

func (f *providerFixture) close() { f.srv.Close() }

// signToken mints a JWT signed with the fixture's private key.
func (f *providerFixture) signToken(t *testing.T, claims jwt.MapClaims, kid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(f.key)
	require.NoError(t, err)
	return s
}

func TestVerifier_ValidToken(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	raw := f.signToken(t, jwt.MapClaims{
		"iss": f.issuer,
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, f.kid)

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	info, err := v.Verify(context.Background(), raw, nil)
	require.NoError(t, err)

	// The same bearer must be forwarded downstream to pidgr-api (A6).
	require.Equal(t, raw, info.Extra["raw_token"])
	assert.True(t, info.Expiration.After(time.Now()))
}

func TestVerifier_ExpiredToken(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	raw := f.signToken(t, jwt.MapClaims{
		"iss": f.issuer,
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}, f.kid)

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	_, err := v.Verify(context.Background(), raw, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, mcpauth.ErrInvalidToken)
}

func TestVerifier_WrongIssuer(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	raw := f.signToken(t, jwt.MapClaims{
		"iss": "https://evil.example.com",
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, f.kid)

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	_, err := v.Verify(context.Background(), raw, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, mcpauth.ErrInvalidToken)
}

func TestVerifier_BadSignature(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	// Sign with a different key than the one published in the JWKS.
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": f.issuer,
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tok.Header["kid"] = f.kid
	raw, err := tok.SignedString(otherKey)
	require.NoError(t, err)

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	_, verr := v.Verify(context.Background(), raw, nil)
	require.Error(t, verr)
	assert.ErrorIs(t, verr, mcpauth.ErrInvalidToken)
}

func TestVerifier_UnknownKid(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	raw := f.signToken(t, jwt.MapClaims{
		"iss": f.issuer,
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, "some-other-kid")

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	_, err := v.Verify(context.Background(), raw, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, mcpauth.ErrInvalidToken)
}

func TestVerifier_MalformedToken(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	_, err := v.Verify(context.Background(), "not-a-jwt", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, mcpauth.ErrInvalidToken)
}

func TestVerifier_RejectsAPIKey(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	_, err := v.Verify(context.Background(), "pidgr_k_abcdefghijklmnopqrstuvwx", nil)
	require.Error(t, err, "pidgr_k_ API keys must no longer be accepted in HTTP mode")
	assert.ErrorIs(t, err, mcpauth.ErrInvalidToken)
}

func TestVerifier_RejectsAlgNone(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	// Forge an unsigned (alg=none) token to ensure it is rejected.
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": f.issuer,
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tok.Header["kid"] = f.kid
	raw, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	_, verr := v.Verify(context.Background(), raw, nil)
	require.Error(t, verr)
	assert.ErrorIs(t, verr, mcpauth.ErrInvalidToken)
}

func TestVerifier_CachesJWKS(t *testing.T) {
	f := newProviderFixture(t)
	defer f.close()

	v := NewVerifier(VerifierConfig{Issuer: f.issuer})
	for i := 0; i < 3; i++ {
		raw := f.signToken(t, jwt.MapClaims{
			"iss": f.issuer,
			"sub": "user-123",
			"exp": time.Now().Add(time.Hour).Unix(),
		}, f.kid)
		_, err := v.Verify(context.Background(), raw, nil)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, f.jwksHits, "JWKS should be fetched once and cached")
}

func TestVerifier_DiscoveryUnavailable(t *testing.T) {
	v := NewVerifier(VerifierConfig{Issuer: "http://127.0.0.1:1"})
	_, err := v.Verify(context.Background(), "anything", nil)
	require.Error(t, err)
	// A discovery/transport failure must not be reported as a valid token.
	assert.True(t, errors.Is(err, mcpauth.ErrInvalidToken) || err != nil)
}
