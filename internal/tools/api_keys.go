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
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

// ── Input types ─────────────────────────────────────────────────────────────

type CreateApiKeyInput struct {
	Name        string   `json:"name" jsonschema:"Human-friendly label (max 200 chars)"`
	Permissions []string `json:"permissions" jsonschema:"Permission names to grant (e.g. PERMISSION_CAMPAIGNS_READ)"`
	ExpiresAt   string   `json:"expires_at,omitempty" jsonschema:"Optional expiration time in RFC 3339 format"`
}

type ListApiKeysInput struct{}

type RevokeApiKeyInput struct {
	ApiKeyID string `json:"api_key_id" jsonschema:"API key UUID to revoke"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerApiKeyTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_api_key",
		Description: "Create a new scoped API key. The full secret is only returned once. Requires ORG_WRITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateApiKeyInput) (*mcp.CallToolResult, any, error) {
		var expiresAt *timestamppb.Timestamp
		if input.ExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, input.ExpiresAt)
			if err != nil {
				r, _ := convert.ErrorResult(err)
				return r, nil, nil
			}
			expiresAt = timestamppb.New(t)
		}
		resp, err := c.ApiKeys.CreateApiKey(ctx, connect.NewRequest(&pidgrv1.CreateApiKeyRequest{
			Name:        input.Name,
			Permissions: toProtoPermissions(input.Permissions),
			ExpiresAt:   expiresAt,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_api_keys",
		Description: "List all active API keys in the organization (metadata only, no secrets). Requires ORG_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListApiKeysInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.ApiKeys.ListApiKeys(ctx, connect.NewRequest(&pidgrv1.ListApiKeysRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "revoke_api_key",
		Description: "Revoke an API key immediately. Requires ORG_WRITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RevokeApiKeyInput) (*mcp.CallToolResult, any, error) {
		_, err := c.ApiKeys.RevokeApiKey(ctx, connect.NewRequest(&pidgrv1.RevokeApiKeyRequest{
			ApiKeyId: input.ApiKeyID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("API key revoked successfully"), nil, nil
	})
}
