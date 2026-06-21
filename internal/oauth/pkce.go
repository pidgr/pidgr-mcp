// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// pkce holds a PKCE code verifier and its derived S256 challenge (RFC 7636).
type pkce struct {
	Verifier  string
	Challenge string
	Method    string
}

// generatePKCE produces a cryptographically random code verifier and its
// base64url-encoded SHA-256 challenge.
func generatePKCE() (pkce, error) {
	// 32 random bytes → 43 base64url chars, within the RFC 7636 43–128 range.
	verifier, err := randomURLSafe(32)
	if err != nil {
		return pkce{}, fmt.Errorf("generate code verifier: %w", err)
	}
	sum := sha256.Sum256([]byte(verifier))
	return pkce{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
		Method:    "S256",
	}, nil
}

// randomState returns a random URL-safe string for CSRF protection.
func randomState() (string, error) {
	s, err := randomURLSafe(24)
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return s, nil
}

func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
