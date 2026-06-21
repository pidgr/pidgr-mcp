// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// loopback is an RFC 8252 native-app redirect receiver. It listens on an
// ephemeral 127.0.0.1 port and captures the authorization code from the
// provider's redirect.
type loopback struct {
	listener net.Listener
	server   *http.Server
	state    string

	result chan loopbackResult
}

type loopbackResult struct {
	code string
	err  error
}

const successPage = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Pidgr</title></head>
<body style="font-family:system-ui,sans-serif;text-align:center;padding-top:4rem">
<h1>Authentication complete</h1>
<p>You can close this tab and return to your terminal.</p>
</body></html>`

// newLoopback binds an ephemeral loopback listener and starts serving the
// /callback route. state is the expected CSRF value.
func newLoopback(state string) (*loopback, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("bind loopback listener: %w", err)
	}

	lb := &loopback{
		listener: ln,
		state:    state,
		result:   make(chan loopbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", lb.handleCallback)
	lb.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() { _ = lb.server.Serve(ln) }()
	return lb, nil
}

// RedirectURI returns the loopback URL the provider should redirect to.
func (l *loopback) RedirectURI() string {
	return fmt.Sprintf("http://%s/callback", l.listener.Addr().String())
}

func (l *loopback) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if errCode := q.Get("error"); errCode != "" {
		desc := q.Get("error_description")
		l.deliver(loopbackResult{err: fmt.Errorf("authorization failed: %s: %s", errCode, desc)})
		http.Error(w, "Authorization failed. You can close this tab.", http.StatusBadRequest)
		return
	}

	if got := q.Get("state"); got != l.state {
		l.deliver(loopbackResult{err: fmt.Errorf("state mismatch: possible CSRF, refusing to continue")})
		http.Error(w, "Invalid state. You can close this tab.", http.StatusBadRequest)
		return
	}

	code := q.Get("code")
	if code == "" {
		l.deliver(loopbackResult{err: fmt.Errorf("authorization response missing code")})
		http.Error(w, "Missing authorization code. You can close this tab.", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(successPage))
	l.deliver(loopbackResult{code: code})
}

// deliver sends the first result and ignores subsequent ones (the channel is
// buffered to size 1).
func (l *loopback) deliver(res loopbackResult) {
	select {
	case l.result <- res:
	default:
	}
}

// Wait blocks until the redirect is received, the context is cancelled, or an
// error occurs.
func (l *loopback) Wait(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-l.result:
		return res.code, res.err
	}
}

// Close shuts down the loopback server.
func (l *loopback) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = l.server.Shutdown(ctx)
}
