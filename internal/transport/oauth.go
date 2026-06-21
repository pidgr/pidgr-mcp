// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package transport

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

// TokenProvider yields a fresh bearer token for an outbound RPC, refreshing or
// re-authenticating as needed. *oauth.Client.AccessToken satisfies this.
type TokenProvider func(ctx context.Context) (string, error)

// NewOAuthClients creates clients that inject a freshly-resolved OAuth access
// token as the Bearer on every request, via the supplied provider.
//
// The IntegrationsService client is co-hosted at baseURL — use
// NewOAuthClientsWithIntegrationsURL when a separate base URL is configured.
func NewOAuthClients(baseURL string, provider TokenProvider) *Clients {
	return NewOAuthClientsWithIntegrationsURL(baseURL, baseURL, provider)
}

// NewOAuthClientsWithIntegrationsURL is the variant that lets the caller route
// IntegrationsService RPCs to a distinct base URL.
func NewOAuthClientsWithIntegrationsURL(baseURL, integrationsURL string, provider TokenProvider) *Clients {
	interceptor := tokenProviderInterceptor(provider)
	opts := connect.WithInterceptors(interceptor)
	return newClients(baseURL, integrationsURL, http.DefaultClient, opts)
}

// tokenProviderInterceptor adds a Bearer token resolved from the provider on
// each request. If the provider fails, the RPC is aborted before dispatch.
func tokenProviderInterceptor(provider TokenProvider) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token, err := provider(ctx)
			if err != nil {
				return nil, fmt.Errorf("resolve oauth access token: %w", err)
			}
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	}
}
