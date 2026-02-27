// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package transport

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/auth"
	pidgrv1connect "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1/pidgrv1connect"
)

// Clients holds Connect-Go clients for all exposed pidgr-api services.
type Clients struct {
	Campaigns     pidgrv1connect.CampaignServiceClient
	Templates     pidgrv1connect.TemplateServiceClient
	Groups        pidgrv1connect.GroupServiceClient
	Teams         pidgrv1connect.TeamServiceClient
	Members       pidgrv1connect.MemberServiceClient
	Organizations pidgrv1connect.OrganizationServiceClient
	Roles         pidgrv1connect.RoleServiceClient
	ApiKeys       pidgrv1connect.ApiKeyServiceClient
	Heatmaps      pidgrv1connect.HeatmapServiceClient
	Replays       pidgrv1connect.ReplayServiceClient
}

// NewStaticTokenClients creates clients that inject a static API key on every request.
// Used for stdio mode where the token comes from an environment variable.
func NewStaticTokenClients(baseURL, apiKey string) *Clients {
	interceptor := staticTokenInterceptor(apiKey)
	opts := connect.WithInterceptors(interceptor)
	return newClients(baseURL, http.DefaultClient, opts)
}

// NewDynamicTokenClients creates clients that extract the JWT from the MCP auth
// context on each request. Used for HTTP mode where the token comes from OAuth.
func NewDynamicTokenClients(baseURL string) *Clients {
	interceptor := dynamicTokenInterceptor()
	opts := connect.WithInterceptors(interceptor)
	return newClients(baseURL, http.DefaultClient, opts)
}

func newClients(baseURL string, httpClient connect.HTTPClient, opts connect.ClientOption) *Clients {
	grpc := connect.WithGRPC()
	return &Clients{
		Campaigns:     pidgrv1connect.NewCampaignServiceClient(httpClient, baseURL, grpc, opts),
		Templates:     pidgrv1connect.NewTemplateServiceClient(httpClient, baseURL, grpc, opts),
		Groups:        pidgrv1connect.NewGroupServiceClient(httpClient, baseURL, grpc, opts),
		Teams:         pidgrv1connect.NewTeamServiceClient(httpClient, baseURL, grpc, opts),
		Members:       pidgrv1connect.NewMemberServiceClient(httpClient, baseURL, grpc, opts),
		Organizations: pidgrv1connect.NewOrganizationServiceClient(httpClient, baseURL, grpc, opts),
		Roles:         pidgrv1connect.NewRoleServiceClient(httpClient, baseURL, grpc, opts),
		ApiKeys:       pidgrv1connect.NewApiKeyServiceClient(httpClient, baseURL, grpc, opts),
		Heatmaps:      pidgrv1connect.NewHeatmapServiceClient(httpClient, baseURL, grpc, opts),
		Replays:       pidgrv1connect.NewReplayServiceClient(httpClient, baseURL, grpc, opts),
	}
}

// staticTokenInterceptor returns an interceptor that adds a static Bearer token header.
func staticTokenInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	}
}

// dynamicTokenInterceptor returns an interceptor that extracts the bearer token
// from the MCP auth context and injects it into the gRPC request.
func dynamicTokenInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if ti := auth.TokenInfoFromContext(ctx); ti != nil {
				if token, ok := ti.Extra["raw_token"].(string); ok {
					req.Header().Set("Authorization", "Bearer "+token)
				}
			}
			return next(ctx, req)
		}
	}
}
