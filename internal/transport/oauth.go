// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

// TokenProvider yields a fresh bearer token for an outbound RPC, refreshing or
// re-authenticating as needed. *oauth.Client.AccessToken satisfies this.
type TokenProvider func(ctx context.Context) (string, error)

// StepUpProvider forces a fresh authorization in response to an RFC 9470
// insufficient_user_authentication challenge and yields the re-authenticated
// token. staleToken is the token the challenge was issued against, so the
// provider can tell whether a concurrent step-up already replaced it.
// *oauth.Client.StepUp satisfies this.
type StepUpProvider func(ctx context.Context, staleToken string) (string, error)

// NewOAuthClients creates clients that inject a freshly-resolved OAuth access
// token as the Bearer on every request, via the supplied provider. stepUp
// answers RFC 9470 insufficient_user_authentication challenges by forcing a
// fresh authorization; pass nil to propagate step-up challenges unhandled.
//
// The IntegrationsService client is co-hosted at baseURL — use
// NewOAuthClientsWithIntegrationsURL when a separate base URL is configured.
func NewOAuthClients(baseURL string, provider TokenProvider, stepUp StepUpProvider) *Clients {
	return NewOAuthClientsWithIntegrationsURL(baseURL, baseURL, provider, stepUp)
}

// NewOAuthClientsWithIntegrationsURL is the variant that lets the caller route
// IntegrationsService RPCs to a distinct base URL.
func NewOAuthClientsWithIntegrationsURL(baseURL, integrationsURL string, provider TokenProvider, stepUp StepUpProvider) *Clients {
	interceptor := tokenProviderInterceptor(provider, stepUp)
	opts := connect.WithInterceptors(interceptor)
	return newClients(baseURL, integrationsURL, http.DefaultClient, opts)
}

// tokenProviderInterceptor adds a Bearer token resolved from the provider on
// each request. If the provider fails, the RPC is aborted before dispatch.
//
// When the server answers with an RFC 9470 step-up challenge — the token is
// valid but its authentication is too old for this operation — the interceptor
// re-authenticates via stepUp and retries the RPC exactly once with the fresh
// token. A second challenge surfaces to the caller instead of looping.
func tokenProviderInterceptor(provider TokenProvider, stepUp StepUpProvider) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token, err := provider(ctx)
			if err != nil {
				return nil, fmt.Errorf("resolve oauth access token: %w", err)
			}
			req.Header().Set("Authorization", "Bearer "+token)
			resp, err := next(ctx, req)
			if err == nil || stepUp == nil || !isStepUpChallenge(err) {
				return resp, err
			}

			fresh, serr := stepUp(ctx, token)
			if serr != nil {
				return nil, fmt.Errorf("step-up re-authentication: %w (challenge: %v)", serr, err)
			}
			req.Header().Set("Authorization", "Bearer "+fresh)
			resp, err = next(ctx, req)
			if err != nil && isStepUpChallenge(err) {
				return nil, fmt.Errorf("the server still requires recent authentication after re-authenticating — please retry, and if it persists check the account's authentication settings: %w", err)
			}
			return resp, err
		}
	}
}

// stepUpErrorCode is the RFC 9470 error code signalling that the token's
// authentication event is older than the server's freshness requirement.
const stepUpErrorCode = "insufficient_user_authentication"

// isStepUpChallenge reports whether err is an RFC 9470 step-up challenge:
// UNAUTHENTICATED carrying the insufficient_user_authentication error code.
// Other 401s (missing/expired/invalid token) must not trigger a forced
// re-authentication.
//
// The code is looked for both in the WWW-Authenticate challenge metadata and
// in the error message: RFC 7230 §4.1.2 forbids authentication header fields
// in HTTP trailers, so over the gRPC protocol — where error metadata travels
// as trailers — the WWW-Authenticate challenge is stripped in transit and only
// the status message survives.
func isStepUpChallenge(err error) bool {
	var cerr *connect.Error
	if !errors.As(err, &cerr) || cerr.Code() != connect.CodeUnauthenticated {
		return false
	}
	if strings.Contains(cerr.Meta().Get("WWW-Authenticate"), `error="`+stepUpErrorCode+`"`) {
		return true
	}
	// Anchored to the message start so a 401 that merely mentions the code
	// somewhere in a diagnostic cannot trigger a forced re-authentication.
	msg := cerr.Message()
	return msg == stepUpErrorCode || strings.HasPrefix(msg, stepUpErrorCode+":")
}
