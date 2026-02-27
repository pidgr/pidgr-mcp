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

type CreateGroupInput struct {
	Name        string `json:"name" jsonschema:"Group name (max 200 chars)"`
	Description string `json:"description,omitempty" jsonschema:"Optional description (max 1000 chars)"`
}

type GetGroupInput struct {
	GroupID string `json:"group_id" jsonschema:"Group UUID"`
}

type ListGroupsInput struct {
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type UpdateGroupInput struct {
	GroupID     string `json:"group_id" jsonschema:"Group UUID to update"`
	Name        string `json:"name,omitempty" jsonschema:"New group name"`
	Description string `json:"description,omitempty" jsonschema:"New description"`
}

type DeleteGroupInput struct {
	GroupID string `json:"group_id" jsonschema:"Group UUID to delete"`
}

type AddGroupMembersInput struct {
	GroupID string   `json:"group_id" jsonschema:"Group UUID"`
	UserIDs []string `json:"user_ids" jsonschema:"User UUIDs to add (max 100)"`
}

type RemoveGroupMembersInput struct {
	GroupID string   `json:"group_id" jsonschema:"Group UUID"`
	UserIDs []string `json:"user_ids" jsonschema:"User UUIDs to remove (max 100)"`
}

type ListGroupMembersInput struct {
	GroupID   string `json:"group_id" jsonschema:"Group UUID"`
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type GetUserGroupMembershipsInput struct {
	UserIDs []string `json:"user_ids" jsonschema:"User UUIDs to look up (max 200)"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerGroupTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_group",
		Description: "Create a new recipient group. Requires GROUPS_WRITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateGroupInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.CreateGroup(ctx, connect.NewRequest(&pidgrv1.CreateGroupRequest{
			Name:        input.Name,
			Description: input.Description,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_group",
		Description: "Retrieve a group by ID. Requires GROUPS_ALL_READ or group membership.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetGroupInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.GetGroup(ctx, connect.NewRequest(&pidgrv1.GetGroupRequest{
			GroupId: input.GroupID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_groups",
		Description: "List groups in the organization with pagination.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListGroupsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.ListGroups(ctx, connect.NewRequest(&pidgrv1.ListGroupsRequest{
			Pagination: &pidgrv1.Pagination{
				PageSize:  input.PageSize,
				PageToken: input.PageToken,
			},
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_group",
		Description: "Update a group's name and/or description. Requires GROUPS_WRITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateGroupInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.UpdateGroup(ctx, connect.NewRequest(&pidgrv1.UpdateGroupRequest{
			GroupId:     input.GroupID,
			Name:        input.Name,
			Description: input.Description,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_group",
		Description: "Delete a group and all its memberships. Default groups cannot be deleted. Requires GROUPS_WRITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteGroupInput) (*mcp.CallToolResult, any, error) {
		_, err := c.Groups.DeleteGroup(ctx, connect.NewRequest(&pidgrv1.DeleteGroupRequest{
			GroupId: input.GroupID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("Group deleted successfully"), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_group_members",
		Description: "Add users to a group (idempotent). Requires GROUPS_WRITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AddGroupMembersInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.AddGroupMembers(ctx, connect.NewRequest(&pidgrv1.AddGroupMembersRequest{
			GroupId: input.GroupID,
			UserIds: input.UserIDs,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "remove_group_members",
		Description: "Remove users from a group (idempotent). Requires GROUPS_WRITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RemoveGroupMembersInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.RemoveGroupMembers(ctx, connect.NewRequest(&pidgrv1.RemoveGroupMembersRequest{
			GroupId: input.GroupID,
			UserIds: input.UserIDs,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_group_members",
		Description: "List members of a group with pagination. Requires GROUPS_ALL_READ or group membership.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListGroupMembersInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.ListGroupMembers(ctx, connect.NewRequest(&pidgrv1.ListGroupMembersRequest{
			GroupId: input.GroupID,
			Pagination: &pidgrv1.Pagination{
				PageSize:  input.PageSize,
				PageToken: input.PageToken,
			},
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_user_group_memberships",
		Description: "Get group memberships for a batch of users. Requires GROUPS_ALL_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetUserGroupMembershipsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Groups.GetUserGroupMemberships(ctx, connect.NewRequest(&pidgrv1.GetUserGroupMembershipsRequest{
			UserIds: input.UserIDs,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
