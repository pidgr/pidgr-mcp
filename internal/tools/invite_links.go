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

type CreateInviteLinkInput struct {
	RoleID               string `json:"role_id,omitempty" jsonschema:"Role UUID to assign to users who join via this link"`
	MaxUses              int32  `json:"max_uses,omitempty" jsonschema:"Maximum number of times the link can be used (0 = unlimited)"`
	ExpiresInHours       int32  `json:"expires_in_hours,omitempty" jsonschema:"Hours until the link expires (0 = no expiry, max 8760)"`
	DataGovernanceRegion string `json:"data_governance_region,omitempty" jsonschema:"Data governance region for users who join via this link (EU, LATAM, BR, APAC, US, or empty for org default)"`
}

type ListInviteLinksInput struct{}

type RevokeInviteLinkInput struct {
	LinkID string `json:"link_id" jsonschema:"Invite link UUID to revoke"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerInviteLinkTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_invite_link",
		Description: "Create a shareable invite link that lets users join the organization. Optionally set a data governance region (EU, LATAM, BR, APAC, US) for users who join via this link.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateInviteLinkInput) (*mcp.CallToolResult, any, error) {
		if err := validateDataGovernanceRegion(input.DataGovernanceRegion); err != nil {
			res, _ := convert.ErrorResult(err)
			return res, nil, nil
		}
		resp, err := c.InviteLinks.CreateInviteLink(ctx, connect.NewRequest(&pidgrv1.CreateInviteLinkRequest{
			RoleId:               input.RoleID,
			MaxUses:              input.MaxUses,
			ExpiresInHours:       input.ExpiresInHours,
			DataGovernanceRegion: input.DataGovernanceRegion,
		}))
		if err != nil {
			res, _ := convert.ErrorResult(err)
			return res, nil, nil
		}
		res, err := convert.ProtoResult(resp.Msg)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_invite_links",
		Description: "List all invite links for the organization.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListInviteLinksInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.InviteLinks.ListInviteLinks(ctx, connect.NewRequest(&pidgrv1.ListInviteLinksRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "revoke_invite_link",
		Description: "Revoke an invite link so it can no longer be used.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RevokeInviteLinkInput) (*mcp.CallToolResult, any, error) {
		_, err := c.InviteLinks.RevokeInviteLink(ctx, connect.NewRequest(&pidgrv1.RevokeInviteLinkRequest{
			InviteLinkId: input.LinkID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("Invite link revoked successfully"), nil, nil
	})
}
