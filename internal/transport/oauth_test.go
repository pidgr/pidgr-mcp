// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	pidgrv1connect "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1/pidgrv1connect"
)

func TestTokenProviderInterceptorInjectsBearer(t *testing.T) {
	interceptor := tokenProviderInterceptor(func(context.Context) (string, error) {
		return "oauth-access-token", nil
	}, nil)

	var captured string
	handler := interceptor(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		captured = req.Header().Get("Authorization")
		return nil, nil
	})

	_, _ = handler(context.Background(), connect.NewRequest(&struct{}{}))

	if captured != "Bearer oauth-access-token" {
		t.Errorf("got Authorization %q, want %q", captured, "Bearer oauth-access-token")
	}
}

func TestTokenProviderInterceptorPropagatesError(t *testing.T) {
	wantErr := errors.New("token unavailable")
	interceptor := tokenProviderInterceptor(func(context.Context) (string, error) {
		return "", wantErr
	}, nil)

	called := false
	handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return nil, nil
	})

	_, err := handler(context.Background(), connect.NewRequest(&struct{}{}))
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped token error, got %v", err)
	}
	if called {
		t.Error("downstream handler must not run when token provider fails")
	}
}

// stepUpChallengeError mirrors the RFC 9470 insufficient_user_authentication
// challenge the resource server sends when a token's authentication is too old.
func stepUpChallengeError() *connect.Error {
	err := connect.NewError(connect.CodeUnauthenticated,
		errors.New("insufficient_user_authentication: this action requires recent authentication"))
	err.Meta().Set("WWW-Authenticate",
		`Bearer error="insufficient_user_authentication", max_age=300`)
	return err
}

func TestStepUpChallengeReauthsAndRetriesOnce(t *testing.T) {
	stepUpCalls := 0
	var staleSeen string
	interceptor := tokenProviderInterceptor(
		func(context.Context) (string, error) { return "stale-token", nil },
		func(_ context.Context, stale string) (string, error) { stepUpCalls++; staleSeen = stale; return "fresh-token", nil },
	)

	var tokens []string
	handler := interceptor(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		tokens = append(tokens, req.Header().Get("Authorization"))
		if len(tokens) == 1 {
			return nil, stepUpChallengeError()
		}
		return nil, nil
	})

	_, err := handler(context.Background(), connect.NewRequest(&struct{}{}))
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if stepUpCalls != 1 {
		t.Errorf("expected exactly 1 step-up re-auth, got %d", stepUpCalls)
	}
	if staleSeen != "stale-token" {
		t.Errorf("step-up must receive the challenged token, got %q", staleSeen)
	}
	want := []string{"Bearer stale-token", "Bearer fresh-token"}
	if len(tokens) != len(want) || tokens[0] != want[0] || tokens[1] != want[1] {
		t.Errorf("got Authorization sequence %v, want %v", tokens, want)
	}
}

// Over the gRPC protocol error metadata travels as HTTP trailers, and RFC 7230
// §4.1.2 forbids WWW-Authenticate in trailers — Go's HTTP/2 server strips it.
// The challenge must therefore also be recognized from the status message alone.
func TestStepUpChallengeDetectedFromMessageWhenHeaderStripped(t *testing.T) {
	headerless := connect.NewError(connect.CodeUnauthenticated,
		errors.New("insufficient_user_authentication: this action requires recent authentication"))

	stepUpCalls := 0
	interceptor := tokenProviderInterceptor(
		func(context.Context) (string, error) { return "stale-token", nil },
		func(context.Context, string) (string, error) { stepUpCalls++; return "fresh-token", nil },
	)

	attempts := 0
	handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		attempts++
		if attempts == 1 {
			return nil, headerless
		}
		return nil, nil
	})

	_, err := handler(context.Background(), connect.NewRequest(&struct{}{}))
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if stepUpCalls != 1 {
		t.Errorf("expected exactly 1 step-up re-auth, got %d", stepUpCalls)
	}
}

func TestStepUpSecondChallengeSurfacesWithoutLooping(t *testing.T) {
	stepUpCalls := 0
	interceptor := tokenProviderInterceptor(
		func(context.Context) (string, error) { return "stale-token", nil },
		func(context.Context, string) (string, error) { stepUpCalls++; return "still-stale-token", nil },
	)

	attempts := 0
	handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		attempts++
		return nil, stepUpChallengeError()
	})

	_, err := handler(context.Background(), connect.NewRequest(&struct{}{}))
	if err == nil {
		t.Fatal("expected the second step-up challenge to surface as an error")
	}
	if stepUpCalls != 1 {
		t.Errorf("expected exactly 1 step-up re-auth (no loop), got %d", stepUpCalls)
	}
	if attempts != 2 {
		t.Errorf("expected exactly 2 attempts (original + one retry), got %d", attempts)
	}
	if !strings.Contains(err.Error(), "recent authentication") {
		t.Errorf("expected a clear step-up failure message, got %q", err.Error())
	}
}

func TestNonStepUpErrorsAreNotRetried(t *testing.T) {
	plainUnauthenticated := connect.NewError(connect.CodeUnauthenticated,
		errors.New("missing authorization header"))
	otherChallenge := connect.NewError(connect.CodeUnauthenticated,
		errors.New("token expired"))
	otherChallenge.Meta().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	mentionsCodeMidMessage := connect.NewError(connect.CodeUnauthenticated,
		errors.New("policy 'insufficient_user_authentication' is not enabled for this resource"))

	cases := []struct {
		name string
		err  error
	}{
		{"unauthenticated without challenge", plainUnauthenticated},
		{"unauthenticated with a different bearer error", otherChallenge},
		{"unauthenticated mentioning the code mid-message", mentionsCodeMidMessage},
		{"permission denied", connect.NewError(connect.CodePermissionDenied, errors.New("missing permission"))},
		{"non-connect error", errors.New("boom")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stepUpCalls := 0
			interceptor := tokenProviderInterceptor(
				func(context.Context) (string, error) { return "token", nil },
				func(context.Context, string) (string, error) { stepUpCalls++; return "fresh", nil },
			)

			attempts := 0
			handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
				attempts++
				return nil, tc.err
			})

			_, err := handler(context.Background(), connect.NewRequest(&struct{}{}))
			if !errors.Is(err, tc.err) {
				t.Errorf("expected the original error to propagate, got %v", err)
			}
			if stepUpCalls != 0 {
				t.Errorf("step-up re-auth must not run for a non-step-up failure, ran %d times", stepUpCalls)
			}
			if attempts != 1 {
				t.Errorf("expected no retry, got %d attempts", attempts)
			}
		})
	}
}

func TestStepUpReauthFailureSurfaces(t *testing.T) {
	reauthErr := errors.New("user closed the browser")
	interceptor := tokenProviderInterceptor(
		func(context.Context) (string, error) { return "stale-token", nil },
		func(context.Context, string) (string, error) { return "", reauthErr },
	)

	attempts := 0
	handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		attempts++
		return nil, stepUpChallengeError()
	})

	_, err := handler(context.Background(), connect.NewRequest(&struct{}{}))
	if !errors.Is(err, reauthErr) {
		t.Errorf("expected the re-auth failure to surface, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected no retry when re-auth fails, got %d attempts", attempts)
	}
}

// stepUpOrgServer rejects any bearer other than the fresh one with the RFC 9470
// challenge, proving the challenge is recognized across a real gRPC round trip
// (where the WWW-Authenticate trailer is stripped and only the status message
// survives) and the retry carries the re-authenticated token.
type stepUpOrgServer struct {
	pidgrv1connect.UnimplementedOrganizationServiceHandler
	attempts int
}

func (s *stepUpOrgServer) GetOrganization(_ context.Context, req *connect.Request[pidgrv1.GetOrganizationRequest]) (*connect.Response[pidgrv1.GetOrganizationResponse], error) {
	s.attempts++
	if req.Header().Get("Authorization") != "Bearer fresh-token" {
		return nil, stepUpChallengeError()
	}
	return connect.NewResponse(&pidgrv1.GetOrganizationResponse{}), nil
}

func TestStepUpChallengeSurvivesGRPCWireAndRetrySucceeds(t *testing.T) {
	server := &stepUpOrgServer{}
	mux := http.NewServeMux()
	mux.Handle(pidgrv1connect.NewOrganizationServiceHandler(server))
	ts := httptest.NewUnstartedServer(mux)
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()

	stepUpCalls := 0
	interceptor := tokenProviderInterceptor(
		func(context.Context) (string, error) { return "stale-token", nil },
		func(context.Context, string) (string, error) { stepUpCalls++; return "fresh-token", nil },
	)
	client := pidgrv1connect.NewOrganizationServiceClient(
		ts.Client(), ts.URL, connect.WithGRPC(), connect.WithInterceptors(interceptor))

	_, err := client.GetOrganization(context.Background(), connect.NewRequest(&pidgrv1.GetOrganizationRequest{}))
	if err != nil {
		t.Fatalf("expected step-up retry to succeed over the wire, got %v", err)
	}
	if stepUpCalls != 1 {
		t.Errorf("expected exactly 1 step-up re-auth, got %d", stepUpCalls)
	}
	if server.attempts != 2 {
		t.Errorf("expected 2 server attempts (challenge + retry), got %d", server.attempts)
	}
}
