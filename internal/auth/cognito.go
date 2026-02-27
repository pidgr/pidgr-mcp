// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

// CognitoVerifier validates Cognito JWTs using JWKS.
type CognitoVerifier struct {
	poolID string
	region string
	issuer string
	jwksURL string

	mu      sync.RWMutex
	keySet  jwk.Set
	fetched bool
}

// NewCognitoVerifier creates a verifier for the given Cognito User Pool.
func NewCognitoVerifier(poolID, region string) *CognitoVerifier {
	issuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", region, poolID)
	return &CognitoVerifier{
		poolID:  poolID,
		region:  region,
		issuer:  issuer,
		jwksURL: issuer + "/.well-known/jwks.json",
	}
}

// Verify implements auth.TokenVerifier for the MCP SDK.
func (v *CognitoVerifier) Verify(ctx context.Context, token string, _ *http.Request) (*mcpauth.TokenInfo, error) {
	keySet, err := v.getKeySet(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch JWKS: %v", mcpauth.ErrInvalidToken, err)
	}

	parsed, err := jwt.Parse([]byte(token), jwt.WithKeySet(keySet), jwt.WithValidate(true))
	if err != nil {
		// If the error is due to unknown kid, try refreshing JWKS once.
		keySet, refreshErr := v.refreshKeySet(ctx)
		if refreshErr != nil {
			return nil, fmt.Errorf("%w: %v", mcpauth.ErrInvalidToken, err)
		}
		parsed, err = jwt.Parse([]byte(token), jwt.WithKeySet(keySet), jwt.WithValidate(true))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", mcpauth.ErrInvalidToken, err)
		}
	}

	// Validate issuer.
	if parsed.Issuer() != v.issuer {
		return nil, fmt.Errorf("%w: invalid issuer: got %q, want %q", mcpauth.ErrInvalidToken, parsed.Issuer(), v.issuer)
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

// getKeySet returns the cached JWKS or fetches it if not yet loaded.
func (v *CognitoVerifier) getKeySet(ctx context.Context, _ string) (jwk.Set, error) {
	v.mu.RLock()
	if v.fetched && v.keySet != nil {
		defer v.mu.RUnlock()
		return v.keySet, nil
	}
	v.mu.RUnlock()
	return v.refreshKeySet(ctx)
}

// refreshKeySet fetches the JWKS from Cognito and updates the cache.
func (v *CognitoVerifier) refreshKeySet(ctx context.Context) (jwk.Set, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	keySet, err := jwk.Fetch(ctx, v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS from %s: %w", v.jwksURL, err)
	}
	v.keySet = keySet
	v.fetched = true
	return keySet, nil
}

// Issuer returns the Cognito issuer URL.
func (v *CognitoVerifier) Issuer() string {
	return v.issuer
}
