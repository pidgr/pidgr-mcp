// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"github.com/pidgr/pidgr-mcp/internal/convert"
	"github.com/pidgr/pidgr-mcp/internal/transport"
)

// ── Input types ─────────────────────────────────────────────────────────────

type CreateSSOProviderInput struct {
	Domain      string `json:"domain" jsonschema:"Email domain (e.g. company.com)"`
	Type        string `json:"type" jsonschema:"Provider type: saml"`
	MetadataURL string `json:"metadata_url" jsonschema:"SAML metadata URL"`
}

type GetSSOProviderInput struct{}

type DeleteSSOProviderInput struct{}

// ── Registration ────────────────────────────────────────────────────────────

func registerSSOTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_sso_provider",
		Description: "Create a SAML SSO provider for the organization.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateSSOProviderInput) (*mcp.CallToolResult, any, error) {
		providerType := pidgrv1.SSOProviderType_SSO_PROVIDER_TYPE_SAML
		if v, ok := pidgrv1.SSOProviderType_value["SSO_PROVIDER_TYPE_"+input.Type]; ok {
			providerType = pidgrv1.SSOProviderType(v)
		}
		resp, err := c.SSO.CreateSSOProvider(ctx, connect.NewRequest(&pidgrv1.CreateSSOProviderRequest{
			Domain:      input.Domain,
			Type:        providerType,
			MetadataUrl: input.MetadataURL,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_sso_provider",
		Description: "Get SSO provider details by UUID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetSSOProviderInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.SSO.GetSSOProvider(ctx, connect.NewRequest(&pidgrv1.GetSSOProviderRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_sso_provider",
		Description: "Delete an SSO provider from the organization.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteSSOProviderInput) (*mcp.CallToolResult, any, error) {
		_, err := c.SSO.DeleteSSOProvider(ctx, connect.NewRequest(&pidgrv1.DeleteSSOProviderRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("SSO provider deleted successfully"), nil, nil
	})
}
