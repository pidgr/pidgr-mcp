// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// refreshLeeway refreshes the access token slightly before it expires to
	// avoid races where a token expires mid-RPC.
	refreshLeeway = 60 * time.Second
	// authTimeout bounds how long we wait for the user to complete the browser
	// flow before giving up.
	authTimeout = 5 * time.Minute
)

// Config configures the OAuth client.
type Config struct {
	// Issuer is the OAuth authorization server base URL. Discovery is fetched
	// from {Issuer}/.well-known/oauth-authorization-server.
	Issuer string
	// ClientID is the static public client ID (e.g. "pidgr-mcp").
	ClientID string
	// Scope is the space-delimited scope string requested.
	Scope string
	// Store persists tokens; defaults to a keychain store with file fallback.
	Store TokenStore
	// Opener opens the browser; defaults to the OS default browser.
	Opener BrowserOpener
	// HTTPClient is used for discovery and token requests; defaults to a client
	// with a sane timeout.
	HTTPClient *http.Client
	// Now returns the current time; defaults to time.Now (injectable for tests).
	Now func() time.Time
}

// Client performs the authorization_code+PKCE+loopback flow and yields fresh
// access tokens, transparently refreshing or re-authenticating as needed.
type Client struct {
	cfg  Config
	hc   *http.Client
	now  func() time.Time
	mu   sync.Mutex
	meta *metadata
}

// NewClient validates the config and returns a ready OAuth client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("oauth: Issuer is required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("oauth: ClientID is required")
	}
	if cfg.Store == nil {
		path, err := defaultStorePath()
		if err != nil {
			return nil, err
		}
		cfg.Store = newKeyringStore(path)
	}
	if cfg.Opener == nil {
		cfg.Opener = openBrowser
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Client{cfg: cfg, hc: cfg.HTTPClient, now: cfg.Now}, nil
}

// AccessToken returns a valid access token, refreshing or re-authenticating as
// needed. It is safe for concurrent use.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored, err := c.cfg.Store.Load()
	if err != nil {
		return "", err
	}

	if stored != nil && !stored.expired(c.now(), refreshLeeway) {
		return stored.AccessToken, nil
	}

	if stored != nil && stored.RefreshToken != "" {
		tok, rerr := c.refresh(ctx, stored.RefreshToken)
		if rerr == nil {
			if serr := c.cfg.Store.Save(tok); serr != nil {
				return "", serr
			}
			return tok.AccessToken, nil
		}
		// Refresh failed (revoked/expired refresh token) — fall through to a
		// fresh browser authorization.
	}

	tok, err := c.authorize(ctx)
	if err != nil {
		return "", err
	}
	if err := c.cfg.Store.Save(tok); err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func (c *Client) metadata(ctx context.Context) (*metadata, error) {
	if c.meta != nil {
		return c.meta, nil
	}
	meta, err := discover(ctx, c.hc, c.cfg.Issuer)
	if err != nil {
		return nil, err
	}
	c.meta = meta
	return meta, nil
}

// authorize runs the full authorization_code+PKCE+loopback browser flow.
func (c *Client) authorize(ctx context.Context) (*Token, error) {
	meta, err := c.metadata(ctx)
	if err != nil {
		return nil, err
	}

	p, err := generatePKCE()
	if err != nil {
		return nil, err
	}
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	lb, err := newLoopback(state)
	if err != nil {
		return nil, err
	}
	defer lb.Close()

	authURL, err := buildAuthorizeURL(meta.AuthorizationEndpoint, authorizeParams{
		clientID:    c.cfg.ClientID,
		redirectURI: lb.RedirectURI(),
		scope:       c.cfg.Scope,
		state:       state,
		challenge:   p.Challenge,
		method:      p.Method,
	})
	if err != nil {
		return nil, err
	}

	if err := c.cfg.Opener(authURL); err != nil {
		return nil, fmt.Errorf("open browser for authorization: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, authTimeout)
	defer cancel()
	code, err := lb.Wait(waitCtx)
	if err != nil {
		return nil, err
	}

	return c.exchangeCode(ctx, meta.TokenEndpoint, code, p.Verifier, lb.RedirectURI())
}

func (c *Client) exchangeCode(ctx context.Context, tokenEndpoint, code, verifier, redirectURI string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	return c.tokenRequest(ctx, tokenEndpoint, form)
}

func (c *Client) refresh(ctx context.Context, refreshToken string) (*Token, error) {
	meta, err := c.metadata(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", c.cfg.ClientID)
	return c.tokenRequest(ctx, meta.TokenEndpoint, form)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`  //nolint:gosec // G117: this is an OAuth token field by design
	RefreshToken string `json:"refresh_token"` //nolint:gosec // G117: this is an OAuth token field by design
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (c *Client) tokenRequest(ctx context.Context, endpoint string, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// G704: the token endpoint comes from operator-configured issuer discovery,
	// not from untrusted request input.
	resp, err := c.hc.Do(req) //nolint:gosec // G704: endpoint is operator-configured, not user-tainted
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	tok := &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
	}
	if tr.ExpiresIn > 0 {
		tok.Expiry = c.now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return tok, nil
}

type authorizeParams struct {
	clientID    string
	redirectURI string
	scope       string
	state       string
	challenge   string
	method      string
}

func buildAuthorizeURL(endpoint string, p authorizeParams) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse authorization endpoint: %w", err)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURI)
	q.Set("code_challenge", p.challenge)
	q.Set("code_challenge_method", p.method)
	q.Set("state", p.state)
	if p.scope != "" {
		q.Set("scope", p.scope)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
