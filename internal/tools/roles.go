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

type ListRolesInput struct{}

type CreateRoleInput struct {
	Name        string   `json:"name" jsonschema:"Role display name (e.g. Team Lead)"`
	Permissions []string `json:"permissions" jsonschema:"Permission names (e.g. PERMISSION_CAMPAIGNS_READ or CAMPAIGNS_READ)"`
}

type UpdateRoleInput struct {
	RoleID      string   `json:"role_id" jsonschema:"Role UUID to update"`
	Name        string   `json:"name,omitempty" jsonschema:"New display name"`
	Permissions []string `json:"permissions,omitempty" jsonschema:"New permission set (replaces existing)"`
}

type DeleteRoleInput struct {
	RoleID string `json:"role_id" jsonschema:"Role UUID to delete"`
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func toProtoPermissions(perms []string) []pidgrv1.Permission {
	result := make([]pidgrv1.Permission, 0, len(perms))
	for _, p := range perms {
		if v, ok := pidgrv1.Permission_value[p]; ok {
			result = append(result, pidgrv1.Permission(v))
		} else if v, ok := pidgrv1.Permission_value["PERMISSION_"+p]; ok {
			result = append(result, pidgrv1.Permission(v))
		}
	}
	return result
}

// ── Registration ────────────────────────────────────────────────────────────

func registerRoleTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_roles",
		Description: "List all roles in the organization with their permission sets. Requires ORG_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListRolesInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Roles.ListRoles(ctx, connect.NewRequest(&pidgrv1.ListRolesRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_role",
		Description: "Create a new custom role with permissions. Requires MEMBERS_MANAGE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateRoleInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Roles.CreateRole(ctx, connect.NewRequest(&pidgrv1.CreateRoleRequest{
			Name:        input.Name,
			Permissions: toProtoPermissions(input.Permissions),
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_role",
		Description: "Update a role's name and/or permissions. System roles cannot be updated. Requires MEMBERS_MANAGE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateRoleInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Roles.UpdateRole(ctx, connect.NewRequest(&pidgrv1.UpdateRoleRequest{
			RoleId:      input.RoleID,
			Name:        input.Name,
			Permissions: toProtoPermissions(input.Permissions),
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_role",
		Description: "Delete a role. Fails if users are assigned to it. System roles cannot be deleted. Requires MEMBERS_MANAGE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteRoleInput) (*mcp.CallToolResult, any, error) {
		_, err := c.Roles.DeleteRole(ctx, connect.NewRequest(&pidgrv1.DeleteRoleRequest{
			RoleId: input.RoleID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("Role deleted successfully"), nil, nil
	})
}
