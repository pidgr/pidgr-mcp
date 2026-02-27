// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

const jwksCacheTTL = time.Hour

// OIDCVerifier validates OIDC JWTs using JWKS discovery.
type OIDCVerifier struct {
	clientID string
	issuer   string
	jwksURL  string

	mu          sync.RWMutex
	keySet      jwk.Set
	fetched     bool
	lastFetched time.Time
}

// NewOIDCVerifier creates a verifier for the given OIDC issuer URL.
// If clientID is non-empty, the aud claim is validated against it.
func NewOIDCVerifier(issuerURL, clientID string) *OIDCVerifier {
	return &OIDCVerifier{
		clientID: clientID,
		issuer:   issuerURL,
		jwksURL:  issuerURL + "/.well-known/jwks.json",
	}
}

// Verify implements auth.TokenVerifier for the MCP SDK.
func (v *OIDCVerifier) Verify(ctx context.Context, token string, _ *http.Request) (*mcpauth.TokenInfo, error) {
	keySet, err := v.getKeySet(ctx)
	if err != nil {
		slog.Warn("JWKS fetch failed", "error", err)
		return nil, fmt.Errorf("%w: token validation failed", mcpauth.ErrInvalidToken)
	}

	parsed, err := jwt.Parse([]byte(token), jwt.WithKeySet(keySet), jwt.WithValidate(true))
	if err != nil {
		// If the error is due to unknown kid, try refreshing JWKS once.
		keySet, refreshErr := v.refreshKeySet(ctx)
		if refreshErr != nil {
			slog.Warn("JWKS refresh failed", "error", refreshErr)
			return nil, fmt.Errorf("%w: token validation failed", mcpauth.ErrInvalidToken)
		}
		parsed, err = jwt.Parse([]byte(token), jwt.WithKeySet(keySet), jwt.WithValidate(true))
		if err != nil {
			slog.Warn("token parse failed after JWKS refresh", "error", err)
			return nil, fmt.Errorf("%w: token validation failed", mcpauth.ErrInvalidToken)
		}
	}

	// Validate issuer.
	if parsed.Issuer() != v.issuer {
		slog.Warn("token issuer mismatch")
		return nil, fmt.Errorf("%w: token validation failed", mcpauth.ErrInvalidToken)
	}

	// Validate audience if client ID is configured.
	if v.clientID != "" {
		aud := parsed.Audience()
		found := false
		for _, a := range aud {
			if a == v.clientID {
				found = true
				break
			}
		}
		if !found {
			slog.Warn("token audience mismatch")
			return nil, fmt.Errorf("%w: token validation failed", mcpauth.ErrInvalidToken)
		}
	}

	// Extract claims.
	sub := parsed.Subject()
	var orgID string
	if claims, ok := parsed.PrivateClaims()["custom:org_id"]; ok {
		orgID, _ = claims.(string)
	}

	exp := parsed.Expiration()
	if exp.IsZero() {
		exp = time.Now().Add(time.Hour) // fallback
	}

	return &mcpauth.TokenInfo{
		Scopes:     []string{"openid", "profile"},
		Expiration: exp,
		UserID:     sub,
		Extra: map[string]any{
			"raw_token": token,
			"sub":       sub,
			"org_id":    orgID,
		},
	}, nil
}

// getKeySet returns the cached JWKS or fetches it if stale or not yet loaded.
func (v *OIDCVerifier) getKeySet(ctx context.Context) (jwk.Set, error) {
	v.mu.RLock()
	if v.fetched && v.keySet != nil && time.Since(v.lastFetched) < jwksCacheTTL {
		defer v.mu.RUnlock()
		return v.keySet, nil
	}
	v.mu.RUnlock()
	return v.refreshKeySet(ctx)
}

// refreshKeySet fetches the JWKS and updates the cache.
func (v *OIDCVerifier) refreshKeySet(ctx context.Context) (jwk.Set, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	keySet, err := jwk.Fetch(ctx, v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch key set: %w", err)
	}
	v.keySet = keySet
	v.fetched = true
	v.lastFetched = time.Now()
	return keySet, nil
}

// Issuer returns the OIDC issuer URL.
func (v *OIDCVerifier) Issuer() string {
	return v.issuer
}
