// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestKeyringStoreRoundTrip(t *testing.T) {
	keyring.MockInit()

	dir := t.TempDir()
	s := newKeyringStore(filepath.Join(dir, "fallback.json"))

	tok := &Token{
		AccessToken:  "kr-at",
		RefreshToken: "kr-rt",
		Expiry:       time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	}
	require.NoError(t, s.Save(tok))

	got, err := s.Load()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "kr-at", got.AccessToken)
	assert.Equal(t, "kr-rt", got.RefreshToken)
}

func TestKeyringStoreLoadEmptyReturnsNil(t *testing.T) {
	keyring.MockInit()

	dir := t.TempDir()
	s := newKeyringStore(filepath.Join(dir, "fallback.json"))

	got, err := s.Load()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestKeyringStoreClear(t *testing.T) {
	keyring.MockInit()

	dir := t.TempDir()
	s := newKeyringStore(filepath.Join(dir, "fallback.json"))

	require.NoError(t, s.Save(&Token{AccessToken: "kr-at"}))
	require.NoError(t, s.Clear())

	got, err := s.Load()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestKeyringStoreFallsBackToFileOnError(t *testing.T) {
	// Simulate an unavailable keychain so Save/Load route to the file fallback.
	keyring.MockInitWithError(keyring.ErrSetDataTooBig)

	dir := t.TempDir()
	path := filepath.Join(dir, "fallback.json")
	s := newKeyringStore(path)

	require.NoError(t, s.Save(&Token{AccessToken: "fb-at", RefreshToken: "fb-rt"}))

	got, err := s.Load()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "fb-at", got.AccessToken)
}
