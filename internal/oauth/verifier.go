// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

// VerifierConfig configures the resource-server JWT verifier used by HTTP mode.
type VerifierConfig struct {
	// Issuer is the OAuth authorization server base URL. The verifier discovers
	// its jwks_uri from {Issuer}/.well-known/oauth-authorization-server and
	// requires the token's `iss` claim to equal this value.
	Issuer string
	// HTTPClient is used for discovery and JWKS fetches; defaults to a client
	// with a sane timeout.
	HTTPClient *http.Client
}

// Verifier validates Pidgr-issued OAuth access tokens (RS256 JWTs) in HTTP mode.
//
// The hosted MCP acts as an OAuth resource server: it verifies the bearer JWT's
// signature against the provider's JWKS, checks `exp` and the `iss` claim, and
// on success forwards the same bearer to pidgr-api, which performs the
// authoritative scope∩principal enforcement (A6). It never accepts pidgr_k_ API
// keys.
type Verifier struct {
	issuer string
	hc     *http.Client

	mu   sync.Mutex
	keys map[string]*rsa.PublicKey // kid -> public key
}

// NewVerifier returns a resource-server token verifier for the configured issuer.
func NewVerifier(cfg VerifierConfig) *Verifier {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Verifier{issuer: cfg.Issuer, hc: hc}
}

// Verify implements the mcpauth.TokenVerifier signature. It returns a TokenInfo
// carrying the original bearer in Extra["raw_token"] for downstream forwarding,
// or an error that unwraps to mcpauth.ErrInvalidToken when the token is invalid.
func (v *Verifier) Verify(ctx context.Context, token string, _ *http.Request) (*mcpauth.TokenInfo, error) {
	keyFunc := func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return v.publicKey(ctx, kid)
	}

	parsed, err := jwt.Parse(
		token,
		keyFunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("%w: %v", mcpauth.ErrInvalidToken, err)
	}

	exp, err := parsed.Claims.GetExpirationTime()
	if err != nil || exp == nil {
		return nil, fmt.Errorf("%w: missing exp", mcpauth.ErrInvalidToken)
	}

	return &mcpauth.TokenInfo{
		Expiration: exp.Time,
		Extra: map[string]any{
			"raw_token": token,
		},
	}, nil
}

// publicKey returns the RSA public key for kid, fetching and caching the JWKS on
// first use. A cache miss for a previously-unseen kid triggers a single refresh
// to pick up rotated keys.
func (v *Verifier) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if key, ok := v.keys[kid]; ok {
		return key, nil
	}
	if err := v.refreshKeysLocked(ctx); err != nil {
		return nil, err
	}
	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("no JWKS key matches kid %q", kid)
	}
	return key, nil
}

func (v *Verifier) refreshKeysLocked(ctx context.Context) error {
	meta, err := discover(ctx, v.hc, v.issuer)
	if err != nil {
		return err
	}
	if meta.JWKSURI == "" {
		return fmt.Errorf("discovery document missing jwks_uri")
	}
	keys, err := fetchJWKS(ctx, v.hc, meta.JWKSURI)
	if err != nil {
		return err
	}
	v.keys = keys
	return nil
}

// jwk is the subset of an RFC 7517 JSON Web Key we consume (RSA signing keys).
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

func fetchJWKS(ctx context.Context, hc *http.Client, jwksURI string) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	// G704: jwks_uri is sourced from operator-configured issuer discovery, not
	// from untrusted request input.
	resp, err := hc.Do(req) //nolint:gosec // G704: jwks_uri is operator-configured, not user-tainted
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("jwks returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var set jwkSet
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		pub, err := rsaPublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("parse jwks key %q: %w", k.Kid, err)
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("jwks contained no usable RSA keys")
	}
	return keys, nil
}

func rsaPublicKey(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}
