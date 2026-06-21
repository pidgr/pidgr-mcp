// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

// Token holds an OAuth access token, its rotating refresh token, and expiry.
type Token struct {
	AccessToken  string    `json:"access_token"`  //nolint:gosec // G117: this is an OAuth token field by design
	RefreshToken string    `json:"refresh_token"` //nolint:gosec // G117: this is an OAuth token field by design
	Expiry       time.Time `json:"expiry"`
}

// expired reports whether the access token is expired or within leeway of
// expiring, relative to now. A zero Expiry is treated as never-expiring.
func (t *Token) expired(now time.Time, leeway time.Duration) bool {
	if t.Expiry.IsZero() {
		return false
	}
	return !now.Add(leeway).Before(t.Expiry)
}

// TokenStore persists OAuth tokens between MCP server invocations.
type TokenStore interface {
	Load() (*Token, error)
	Save(*Token) error
	Clear() error
}

const (
	keyringService = "pidgr-mcp"
	keyringUser    = "oauth-token"
)

// fileStore persists tokens to a 0600 JSON file.
type fileStore struct {
	path string
}

func newFileStore(path string) *fileStore {
	return &fileStore{path: path}
}

func (s *fileStore) Load() (*Token, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read token file: %w", err)
	}
	return unmarshalToken(data)
}

func (s *fileStore) Save(t *Token) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}

func (s *fileStore) Clear() error {
	err := os.Remove(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove token file: %w", err)
	}
	return nil
}

// keyringStore persists tokens in the OS keychain, falling back to a file store
// when the keychain is unavailable (headless CI, locked keyring, etc).
type keyringStore struct {
	fallback *fileStore
}

func newKeyringStore(fallbackPath string) *keyringStore {
	return &keyringStore{fallback: newFileStore(fallbackPath)}
}

func (s *keyringStore) Load() (*Token, error) {
	secret, err := keyring.Get(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return s.fallback.Load()
	}
	return unmarshalToken([]byte(secret))
}

func (s *keyringStore) Save(t *Token) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := keyring.Set(keyringService, keyringUser, string(data)); err != nil {
		return s.fallback.Save(t)
	}
	return nil
}

func (s *keyringStore) Clear() error {
	err := keyring.Delete(keyringService, keyringUser)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		// Best effort; also clear the fallback.
		_ = s.fallback.Clear()
		return nil
	}
	return s.fallback.Clear()
}

func unmarshalToken(data []byte) (*Token, error) {
	var t Token
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return &t, nil
}

// defaultStorePath returns the file-fallback path under the user's config dir.
func defaultStorePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "pidgr", "oauth-token.json"), nil
}
