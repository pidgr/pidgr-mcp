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

type ListDevicesInput struct{}

type ListMemberDevicesInput struct {
	UserID string `json:"user_id" jsonschema:"User UUID to list devices for"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerDeviceTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_devices",
		Description: "List all registered push notification devices for the current user.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListDevicesInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Devices.ListDevices(ctx, connect.NewRequest(&pidgrv1.ListDevicesRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_member_devices",
		Description: "List registered devices for a specific user. Use list_users to find user UUIDs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListMemberDevicesInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Devices.ListMemberDevices(ctx, connect.NewRequest(&pidgrv1.ListMemberDevicesRequest{
			UserId: input.UserID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
