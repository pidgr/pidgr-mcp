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

// Clients holds the Connect-Go clients for the Pidgr API services this MCP server wraps.
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
	Privacy       pidgrv1connect.PrivacyServiceClient
	Audit         pidgrv1connect.AuditServiceClient
	SSO           pidgrv1connect.SSOServiceClient
	InviteLinks   pidgrv1connect.InviteLinkServiceClient
	Devices       pidgrv1connect.DeviceServiceClient
	Insights      pidgrv1connect.InsightsServiceClient
	// Integrations is the Connect-Go client for pidgr.v1.IntegrationsService
	// (reachability registry, region policy, cost-cap, dispatch worker). The
	// base URL is configured via PIDGR_INTEGRATIONS_URL with a fallback to
	// PIDGR_API_URL when unset — useful when the IntegrationsService is
	// co-hosted at the same gRPC ingress as the main API.
	Integrations pidgrv1connect.IntegrationsServiceClient
}

// NewDynamicTokenClients creates clients that extract the bearer token from the MCP
// auth context on each request. Used for HTTP mode where the OAuth bearer is
// verified by the RequireBearerToken middleware.
//
// The IntegrationsService client is co-hosted at baseURL — use
// NewDynamicTokenClientsWithIntegrationsURL when a separate base URL is configured.
func NewDynamicTokenClients(baseURL string) *Clients {
	return NewDynamicTokenClientsWithIntegrationsURL(baseURL, baseURL)
}

// NewDynamicTokenClientsWithIntegrationsURL is the variant that lets the
// caller route IntegrationsService RPCs to a distinct base URL.
func NewDynamicTokenClientsWithIntegrationsURL(baseURL, integrationsURL string) *Clients {
	interceptor := dynamicTokenInterceptor()
	opts := connect.WithInterceptors(interceptor)
	return newClients(baseURL, integrationsURL, http.DefaultClient, opts)
}

func newClients(baseURL, integrationsURL string, httpClient connect.HTTPClient, opts connect.ClientOption) *Clients {
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
		Privacy:       pidgrv1connect.NewPrivacyServiceClient(httpClient, baseURL, grpc, opts),
		Audit:         pidgrv1connect.NewAuditServiceClient(httpClient, baseURL, grpc, opts),
		SSO:           pidgrv1connect.NewSSOServiceClient(httpClient, baseURL, grpc, opts),
		InviteLinks:   pidgrv1connect.NewInviteLinkServiceClient(httpClient, baseURL, grpc, opts),
		Devices:       pidgrv1connect.NewDeviceServiceClient(httpClient, baseURL, grpc, opts),
		Insights:      pidgrv1connect.NewInsightsServiceClient(httpClient, baseURL, grpc, opts),
		Integrations:  pidgrv1connect.NewIntegrationsServiceClient(httpClient, integrationsURL, grpc, opts),
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
