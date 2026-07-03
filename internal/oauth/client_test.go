// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memStore is an in-memory token store for tests.
type memStore struct {
	mu  sync.Mutex
	tok *Token
}

func (m *memStore) Load() (*Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tok, nil
}

func (m *memStore) Save(t *Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t
	m.tok = &cp
	return nil
}

func (m *memStore) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tok = nil
	return nil
}

// fakeProvider stands in for the OAuth authorization server. It serves
// discovery and a token endpoint and records what it received.
type fakeProvider struct {
	srv *httptest.Server

	mu               sync.Mutex
	tokenForms       []url.Values
	accessToken      string
	refreshToken     string
	expiresIn        int
	refreshShouldErr bool
}

func newFakeProvider() *fakeProvider {
	f := &fakeProvider{
		accessToken:  "access-1",
		refreshToken: "refresh-1",
		expiresIn:    3600,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		base := f.srv.URL
		_, _ = w.Write([]byte(`{
			"issuer": "` + base + `",
			"authorization_endpoint": "` + base + `/authorize",
			"token_endpoint": "` + base + `/token",
			"code_challenge_methods_supported": ["S256"]
		}`))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.tokenForms = append(f.tokenForms, r.PostForm)
		grant := r.PostForm.Get("grant_type")
		if grant == "refresh_token" && f.refreshShouldErr {
			f.mu.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		at, rt, exp := f.accessToken, f.refreshToken, f.expiresIn
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		body := `{"access_token":"` + at + `","refresh_token":"` + rt +
			`","token_type":"Bearer","expires_in":` + itoa(exp) + `}`
		_, _ = w.Write([]byte(body))
	})
	f.srv = httptest.NewServer(mux)
	return f
}

func (f *fakeProvider) Close() { f.srv.Close() }

func (f *fakeProvider) forms() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]url.Values, len(f.tokenForms))
	copy(out, f.tokenForms)
	return out
}

func (f *fakeProvider) setTokens(at, rt string, exp int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.accessToken, f.refreshToken, f.expiresIn = at, rt, exp
}

func (f *fakeProvider) failRefresh(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshShouldErr = v
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}

// drivingBrowser simulates a user completing the consent flow: it parses the
// authorize URL the client tried to open, then drives the loopback redirect
// with a fixed code echoing the provided state.
func drivingBrowser(t *testing.T, code string) BrowserOpener {
	return func(authorizeURL string) error {
		u, err := url.Parse(authorizeURL)
		if err != nil {
			return err
		}
		redirect := u.Query().Get("redirect_uri")
		state := u.Query().Get("state")
		require.NotEmpty(t, redirect)
		require.NotEmpty(t, state)
		go func() {
			resp, derr := http.Get(redirect + "?code=" + code + "&state=" + url.QueryEscape(state))
			if derr == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

func newTestClient(t *testing.T, f *fakeProvider, store TokenStore, opener BrowserOpener, now func() time.Time) *Client {
	t.Helper()
	c, err := NewClient(Config{
		Issuer:     f.srv.URL,
		ClientID:   "pidgr-mcp",
		Scope:      "openid",
		Store:      store,
		Opener:     opener,
		HTTPClient: f.srv.Client(),
		Now:        now,
	})
	require.NoError(t, err)
	return c
}

func TestClientFullBrowserFlow(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	store := &memStore{}

	c := newTestClient(t, f, store, drivingBrowser(t, "auth-code-xyz"), time.Now)

	tok, err := c.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "access-1", tok)

	forms := f.forms()
	require.Len(t, forms, 1)
	assert.Equal(t, "authorization_code", forms[0].Get("grant_type"))
	assert.Equal(t, "auth-code-xyz", forms[0].Get("code"))
	assert.Equal(t, "pidgr-mcp", forms[0].Get("client_id"))
	assert.NotEmpty(t, forms[0].Get("code_verifier"), "PKCE verifier must be sent in exchange")
	assert.Contains(t, forms[0].Get("redirect_uri"), "127.0.0.1")

	// Tokens persisted.
	saved, _ := store.Load()
	require.NotNil(t, saved)
	assert.Equal(t, "access-1", saved.AccessToken)
	assert.Equal(t, "refresh-1", saved.RefreshToken)
}

func TestClientReusesStoredValidToken(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	store := &memStore{}
	_ = store.Save(&Token{
		AccessToken:  "cached-at",
		RefreshToken: "cached-rt",
		Expiry:       time.Now().Add(time.Hour),
	})

	// Opener that fails the test if the browser is opened.
	opener := func(string) error {
		t.Fatal("browser must not open when a valid token is cached")
		return nil
	}
	c := newTestClient(t, f, store, opener, time.Now)

	tok, err := c.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "cached-at", tok)
	assert.Empty(t, f.forms(), "token endpoint must not be called when cache is valid")
}

func TestClientRefreshesExpiredToken(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	f.setTokens("access-2", "refresh-2", 3600) // rotated tokens

	store := &memStore{}
	_ = store.Save(&Token{
		AccessToken:  "stale-at",
		RefreshToken: "stale-rt",
		Expiry:       time.Now().Add(-time.Hour), // expired
	})

	opener := func(string) error {
		t.Fatal("browser must not open when refresh succeeds")
		return nil
	}
	c := newTestClient(t, f, store, opener, time.Now)

	tok, err := c.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "access-2", tok)

	forms := f.forms()
	require.Len(t, forms, 1)
	assert.Equal(t, "refresh_token", forms[0].Get("grant_type"))
	assert.Equal(t, "stale-rt", forms[0].Get("refresh_token"))

	// Rotated refresh token persisted.
	saved, _ := store.Load()
	require.NotNil(t, saved)
	assert.Equal(t, "access-2", saved.AccessToken)
	assert.Equal(t, "refresh-2", saved.RefreshToken, "rotated refresh token must be stored")
}

func TestClientReauthsWhenRefreshFails(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	f.failRefresh(true)
	f.setTokens("access-3", "refresh-3", 3600)

	store := &memStore{}
	_ = store.Save(&Token{
		AccessToken:  "stale-at",
		RefreshToken: "bad-rt",
		Expiry:       time.Now().Add(-time.Hour),
	})

	c := newTestClient(t, f, store, drivingBrowser(t, "reauth-code"), time.Now)

	tok, err := c.AccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "access-3", tok)

	forms := f.forms()
	require.Len(t, forms, 2)
	assert.Equal(t, "refresh_token", forms[0].Get("grant_type"))     // failed attempt
	assert.Equal(t, "authorization_code", forms[1].Get("grant_type")) // re-auth fallback
	assert.Equal(t, "reauth-code", forms[1].Get("code"))
}

// capturingBrowser records the authorize URL before driving the consent flow,
// so tests can assert on the authorization request parameters.
func capturingBrowser(t *testing.T, code string, captured *string) BrowserOpener {
	inner := drivingBrowser(t, code)
	return func(authorizeURL string) error {
		*captured = authorizeURL
		return inner(authorizeURL)
	}
}

func TestStepUpForcesFreshAuthorizationWithMaxAgeZero(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	f.setTokens("stepped-up-at", "stepped-up-rt", 3600)

	// A valid cached token must NOT satisfy a step-up: the whole point is that
	// the server judged its authentication too old.
	store := &memStore{}
	_ = store.Save(&Token{
		AccessToken:  "cached-at",
		RefreshToken: "cached-rt",
		Expiry:       time.Now().Add(time.Hour),
	})

	var authorizeURL string
	c := newTestClient(t, f, store, capturingBrowser(t, "stepup-code", &authorizeURL), time.Now)

	tok, err := c.StepUp(context.Background(), "cached-at")
	require.NoError(t, err)
	assert.Equal(t, "stepped-up-at", tok)

	u, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	assert.Equal(t, "0", u.Query().Get("max_age"), "step-up must send max_age=0 to force fresh authentication")

	forms := f.forms()
	require.Len(t, forms, 1)
	assert.Equal(t, "authorization_code", forms[0].Get("grant_type"), "step-up must run the full authorization_code flow, not a refresh")

	// The cached token pair is replaced so subsequent calls use the fresh one.
	saved, _ := store.Load()
	require.NotNil(t, saved)
	assert.Equal(t, "stepped-up-at", saved.AccessToken)
	assert.Equal(t, "stepped-up-rt", saved.RefreshToken)
}

func TestStepUpReusesTokenMintedByConcurrentStepUp(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	f.setTokens("fresh-at", "fresh-rt", 3600)

	store := &memStore{}
	opens := 0
	inner := drivingBrowser(t, "stepup-code")
	opener := func(u string) error {
		opens++
		return inner(u)
	}
	c := newTestClient(t, f, store, opener, time.Now)

	tok1, err := c.StepUp(context.Background(), "stale-at")
	require.NoError(t, err)
	assert.Equal(t, "fresh-at", tok1)
	assert.Equal(t, 1, opens)

	// A second challenge against the same stale token (a racing tool call)
	// must reuse the token the first step-up just minted, not prompt again.
	tok2, err := c.StepUp(context.Background(), "stale-at")
	require.NoError(t, err)
	assert.Equal(t, "fresh-at", tok2)
	assert.Equal(t, 1, opens, "the freshly stepped-up token must be reused without a second browser flow")
}

func TestStepUpOnTheSteppedUpTokenPromptsAgain(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	f.setTokens("fresh-1", "rt-1", 3600)

	store := &memStore{}
	opens := 0
	inner := drivingBrowser(t, "stepup-code")
	opener := func(u string) error {
		opens++
		return inner(u)
	}
	c := newTestClient(t, f, store, opener, time.Now)

	_, err := c.StepUp(context.Background(), "stale-at")
	require.NoError(t, err)

	// A challenge against the step-up-minted token itself means that token is
	// no longer fresh enough — it must not be reused.
	f.setTokens("fresh-2", "rt-2", 3600)
	tok, err := c.StepUp(context.Background(), "fresh-1")
	require.NoError(t, err)
	assert.Equal(t, "fresh-2", tok)
	assert.Equal(t, 2, opens)
}

func TestStepUpDoesNotReuseRefreshMintedToken(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()
	f.setTokens("fresh-at", "fresh-rt", 3600)

	// The store holds a token that differs from the challenged one but was NOT
	// minted by a step-up (e.g. a routine refresh rotated it). A refresh does
	// not renew the authentication event, so it cannot satisfy the challenge.
	store := &memStore{}
	_ = store.Save(&Token{
		AccessToken:  "refresh-rotated-at",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(time.Hour),
	})

	opens := 0
	inner := drivingBrowser(t, "stepup-code")
	opener := func(u string) error {
		opens++
		return inner(u)
	}
	c := newTestClient(t, f, store, opener, time.Now)

	tok, err := c.StepUp(context.Background(), "stale-at")
	require.NoError(t, err)
	assert.Equal(t, "fresh-at", tok)
	assert.Equal(t, 1, opens, "a refresh-rotated token must not short-circuit the step-up")
}

func TestRegularAuthorizationOmitsMaxAge(t *testing.T) {
	f := newFakeProvider()
	defer f.Close()

	var authorizeURL string
	c := newTestClient(t, f, &memStore{}, capturingBrowser(t, "auth-code", &authorizeURL), time.Now)

	_, err := c.AccessToken(context.Background())
	require.NoError(t, err)

	u, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	assert.False(t, u.Query().Has("max_age"), "a non-step-up authorization must not request forced re-authentication")
}
