// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePKCE(t *testing.T) {
	p, err := generatePKCE()
	require.NoError(t, err)

	assert.Equal(t, "S256", p.Method)
	assert.GreaterOrEqual(t, len(p.Verifier), 43, "verifier must be at least 43 chars (RFC 7636)")
	assert.LessOrEqual(t, len(p.Verifier), 128, "verifier must be at most 128 chars (RFC 7636)")

	// Verifier must use only the unreserved URL-safe alphabet.
	assert.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9._~-]+$`), p.Verifier)

	// Challenge must be the base64url(SHA256(verifier)), no padding.
	sum := sha256.Sum256([]byte(p.Verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	assert.Equal(t, want, p.Challenge)
	assert.NotContains(t, p.Challenge, "=", "challenge must be unpadded base64url")
}

func TestGeneratePKCEUnique(t *testing.T) {
	a, err := generatePKCE()
	require.NoError(t, err)
	b, err := generatePKCE()
	require.NoError(t, err)
	assert.NotEqual(t, a.Verifier, b.Verifier)
	assert.NotEqual(t, a.Challenge, b.Challenge)
}

func TestRandomState(t *testing.T) {
	a, err := randomState()
	require.NoError(t, err)
	b, err := randomState()
	require.NoError(t, err)

	assert.NotEmpty(t, a)
	assert.NotEqual(t, a, b)
	assert.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9._~-]+$`), a)
}
