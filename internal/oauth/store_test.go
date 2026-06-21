// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := newFileStore(filepath.Join(dir, "tokens.json"))

	tok := &Token{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		Expiry:       time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	}
	require.NoError(t, s.Save(tok))

	got, err := s.Load()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tok.AccessToken, got.AccessToken)
	assert.Equal(t, tok.RefreshToken, got.RefreshToken)
	assert.True(t, tok.Expiry.Equal(got.Expiry))
}

func TestFileStoreLoadMissingReturnsNil(t *testing.T) {
	dir := t.TempDir()
	s := newFileStore(filepath.Join(dir, "does-not-exist.json"))

	got, err := s.Load()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFileStorePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	s := newFileStore(path)

	require.NoError(t, s.Save(&Token{AccessToken: "at"}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestFileStoreClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	s := newFileStore(path)

	require.NoError(t, s.Save(&Token{AccessToken: "at"}))
	require.NoError(t, s.Clear())

	got, err := s.Load()
	require.NoError(t, err)
	assert.Nil(t, got)

	// Clearing an already-absent store is not an error.
	require.NoError(t, s.Clear())
}

func TestTokenExpired(t *testing.T) {
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)

	assert.False(t, (&Token{Expiry: now.Add(10 * time.Minute)}).expired(now, time.Minute))
	assert.True(t, (&Token{Expiry: now.Add(30 * time.Second)}).expired(now, time.Minute))
	assert.True(t, (&Token{Expiry: now.Add(-time.Second)}).expired(now, time.Minute))
	// Zero expiry is treated as never-expiring (defensive).
	assert.False(t, (&Token{}).expired(now, time.Minute))
}
