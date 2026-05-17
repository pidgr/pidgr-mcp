// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"github.com/pidgr/pidgr-mcp/internal/convert"
	"github.com/pidgr/pidgr-mcp/internal/transport"
)

// ── Input types ─────────────────────────────────────────────────────────────

type ListReachabilitiesForUserInput struct {
	OrgID  string `json:"org_id" jsonschema:"Organization UUID owning the (user, channel) reachability rows"`
	UserID string `json:"user_id" jsonschema:"User UUID to list per-channel reachability metadata for"`
}

type UpsertReachabilityInput struct {
	OrgID  string `json:"org_id" jsonschema:"Organization UUID this reachability entry belongs to"`
	UserID string `json:"user_id" jsonschema:"User UUID this reachability entry is for"`
	// Channel is a canonical ChannelName enum-name string (e.g. CHANNEL_NAME_EMAIL,
	// CHANNEL_NAME_SLACK). Unknown values return an invalid-argument error and the
	// upstream RPC is not invoked.
	Channel string `json:"channel" jsonschema:"Channel enum name, e.g. CHANNEL_NAME_EMAIL / CHANNEL_NAME_SMS / CHANNEL_NAME_SLACK / CHANNEL_NAME_TELEGRAM / CHANNEL_NAME_WHATSAPP / CHANNEL_NAME_MICROSOFT_TEAMS / CHANNEL_NAME_LINE / CHANNEL_NAME_WEBHOOK"`
	// IdentifierPlaintext is sensitive (email address, phone number, Slack user
	// ID, etc.). The upstream service KMS-encrypts at rest and computes the
	// HMAC lookup hash. pidgr-mcp NEVER logs this value.
	IdentifierPlaintext string  `json:"identifier_plaintext" jsonschema:"Plaintext channel identifier (email, phone, Slack user id, etc). SENSITIVE — encrypted server-side, never logged."`
	RegionConstraint    *string `json:"region_constraint,omitempty" jsonschema:"Optional AWS region (e.g. eu-west-1) the user's data must remain in"`
}

type RemoveReachabilityInput struct {
	OrgID   string `json:"org_id" jsonschema:"Organization UUID owning the row"`
	UserID  string `json:"user_id" jsonschema:"User UUID this reachability entry is for"`
	Channel string `json:"channel" jsonschema:"Channel enum name (e.g. CHANNEL_NAME_EMAIL)"`
}

type GetCostCapPolicyInput struct {
	OrgID   string `json:"org_id" jsonschema:"Organization UUID to read the cost-cap state for"`
	Channel string `json:"channel" jsonschema:"Channel enum name (e.g. CHANNEL_NAME_EMAIL). Returns the channel default cap when no per-period row exists — never NOT_FOUND."`
}

type SetCostCapPolicyInput struct {
	OrgID     string `json:"org_id" jsonschema:"Organization UUID to write the cost cap for"`
	Channel   string `json:"channel" jsonschema:"Channel enum name (e.g. CHANNEL_NAME_EMAIL)"`
	CapMicros int64  `json:"cap_micros" jsonschema:"Calendar-month cost cap in micros (1/1_000_000 USD). Admin-only."`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerIntegrationsTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_reachabilities_for_user",
		Description: "List per-channel reachability metadata for a single (org, user) pair. Returns one entry per ChannelName configured. Plaintext identifiers and envelope ciphertext are never returned — only metadata.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input ListReachabilitiesForUserInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Integrations.ListReachabilityForUser(ctx, connect.NewRequest(&pidgrv1.ListReachabilityForUserRequest{
			OrgId:  input.OrgID,
			UserId: input.UserID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "upsert_reachability",
		Description: "Record a recipient identifier for a (user, channel) tuple. " +
			"The `identifier_plaintext` argument is sensitive (email address, phone number, Slack user id, etc.) — it is column-level KMS-encrypted server-side and never logged. " +
			"The server computes the HMAC lookup hash; clients pass the plaintext as-is. " +
			"Use `list_users` to resolve the user UUID and `list_reachabilities_for_user` to inspect existing entries.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input UpsertReachabilityInput) (*mcp.CallToolResult, any, error) {
		channel, err := parseChannelName(input.Channel)
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		resp, err := c.Integrations.UpsertReachability(ctx, connect.NewRequest(&pidgrv1.UpsertReachabilityRequest{
			OrgId:               input.OrgID,
			UserId:              input.UserID,
			Channel:             channel,
			IdentifierPlaintext: input.IdentifierPlaintext,
			RegionConstraint:    input.RegionConstraint,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "remove_reachability",
		Description: "Idempotently remove a reachability identifier for a (user, channel) tuple. Returns `removed: true` when a row existed, `removed: false` otherwise — never NOT_FOUND. Emits a REACHABILITY_REMOVE audit row server-side before the delete commits.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input RemoveReachabilityInput) (*mcp.CallToolResult, any, error) {
		channel, err := parseChannelName(input.Channel)
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		resp, err := c.Integrations.RemoveReachability(ctx, connect.NewRequest(&pidgrv1.RemoveReachabilityRequest{
			OrgId:   input.OrgID,
			UserId:  input.UserID,
			Channel: channel,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_cost_cap_policy",
		Description: "Read the current calendar-month cost-cap state (cap_micros, used_micros, period_yyyymm) for an (org, channel). Returns the channel default cap when no row exists — never NOT_FOUND.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input GetCostCapPolicyInput) (*mcp.CallToolResult, any, error) {
		channel, err := parseChannelName(input.Channel)
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		resp, err := c.Integrations.GetCostCapPolicy(ctx, connect.NewRequest(&pidgrv1.GetCostCapPolicyRequest{
			OrgId:   input.OrgID,
			Channel: channel,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_cost_cap_policy",
		Description: "Admin-only upsert of the current calendar-month cost cap (micros) for an (org, channel). Future periods inherit this value until the next call.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input SetCostCapPolicyInput) (*mcp.CallToolResult, any, error) {
		channel, err := parseChannelName(input.Channel)
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		resp, err := c.Integrations.SetCostCapPolicy(ctx, connect.NewRequest(&pidgrv1.SetCostCapPolicyRequest{
			OrgId:     input.OrgID,
			Channel:   channel,
			CapMicros: input.CapMicros,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}

// parseChannelName parses a ChannelName enum-name string (e.g. "CHANNEL_NAME_EMAIL")
// into the typed enum. Returns a Connect INVALID_ARGUMENT error for unknown names
// and for the UNSPECIFIED sentinel so callers can't bypass channel selection.
func parseChannelName(name string) (pidgrv1.ChannelName, error) {
	v, ok := pidgrv1.ChannelName_value[name]
	if !ok {
		return pidgrv1.ChannelName_CHANNEL_NAME_UNSPECIFIED,
			connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown channel %q", name))
	}
	if pidgrv1.ChannelName(v) == pidgrv1.ChannelName_CHANNEL_NAME_UNSPECIFIED {
		return pidgrv1.ChannelName_CHANNEL_NAME_UNSPECIFIED,
			connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("channel must be set"))
	}
	return pidgrv1.ChannelName(v), nil
}
