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

type UserProfileInput struct {
	FirstName        string            `json:"first_name,omitempty" jsonschema:"Given name"`
	LastName         string            `json:"last_name,omitempty" jsonschema:"Family name"`
	Department       string            `json:"department,omitempty" jsonschema:"Department or team"`
	Title            string            `json:"title,omitempty" jsonschema:"Job title"`
	Phone            string            `json:"phone,omitempty" jsonschema:"Phone number"`
	Location         string            `json:"location,omitempty" jsonschema:"Office or location"`
	EmployeeID       string            `json:"employee_id,omitempty" jsonschema:"Organization employee ID"`
	ManagerName      string            `json:"manager_name,omitempty" jsonschema:"Direct manager name"`
	StartDate        string            `json:"start_date,omitempty" jsonschema:"Employment start date (YYYY-MM-DD)"`
	CustomAttributes map[string]string `json:"custom_attributes,omitempty" jsonschema:"Custom profile attributes"`
}

type InviteUserInput struct {
	Email   string            `json:"email" jsonschema:"Email address to invite (max 254 chars)"`
	Name    string            `json:"name" jsonschema:"Display name (max 200 chars)"`
	RoleID  string            `json:"role_id,omitempty" jsonschema:"Role UUID to assign (defaults to employee role)"`
	Profile *UserProfileInput `json:"profile,omitempty" jsonschema:"Optional profile attributes to pre-fill"`
}

type GetUserInput struct {
	UserID string `json:"user_id" jsonschema:"User UUID to retrieve"`
}

type ListUsersInput struct {
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type UpdateUserRoleInput struct {
	UserID string `json:"user_id" jsonschema:"User UUID"`
	RoleID string `json:"role_id" jsonschema:"New role UUID to assign"`
}

type DeactivateUserInput struct {
	UserID string `json:"user_id" jsonschema:"User UUID to deactivate"`
}

type UpdateUserProfileInput struct {
	UserID  string           `json:"user_id" jsonschema:"User UUID to update"`
	Profile UserProfileInput `json:"profile" jsonschema:"Profile attributes to set"`
}

// ── Registration ────────────────────────────────────────────────────────────

func toProtoProfile(p *UserProfileInput) *pidgrv1.UserProfile {
	if p == nil {
		return nil
	}
	return &pidgrv1.UserProfile{
		FirstName:        p.FirstName,
		LastName:         p.LastName,
		Department:       p.Department,
		Title:            p.Title,
		Phone:            p.Phone,
		Location:         p.Location,
		EmployeeId:       p.EmployeeID,
		ManagerName:      p.ManagerName,
		StartDate:        p.StartDate,
		CustomAttributes: p.CustomAttributes,
	}
}

func registerMemberTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "invite_user",
		Description: "Invite a new user to the organization via email. Requires MEMBERS_INVITE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input InviteUserInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Members.InviteUser(ctx, connect.NewRequest(&pidgrv1.InviteUserRequest{
			Email:   input.Email,
			Name:    input.Name,
			RoleId:  input.RoleID,
			Profile: toProtoProfile(input.Profile),
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_user",
		Description: "Retrieve a user by ID. Requires MEMBERS_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetUserInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Members.GetUser(ctx, connect.NewRequest(&pidgrv1.GetUserRequest{
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
		Name:        "list_users",
		Description: "List all users in the organization with pagination. Requires MEMBERS_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListUsersInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Members.ListUsers(ctx, connect.NewRequest(&pidgrv1.ListUsersRequest{
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
		Name:        "update_user_role",
		Description: "Change a user's role. Requires MEMBERS_MANAGE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateUserRoleInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Members.UpdateUserRole(ctx, connect.NewRequest(&pidgrv1.UpdateUserRoleRequest{
			UserId: input.UserID,
			RoleId: input.RoleID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "deactivate_user",
		Description: "Deactivate a user (they will no longer receive messages). Requires MEMBERS_MANAGE permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeactivateUserInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Members.DeactivateUser(ctx, connect.NewRequest(&pidgrv1.DeactivateUserRequest{
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
		Name:        "update_user_profile",
		Description: "Update a user's profile attributes (department, title, etc.). Requires MEMBERS_MANAGE permission for other users.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateUserProfileInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Members.UpdateUserProfile(ctx, connect.NewRequest(&pidgrv1.UpdateUserProfileRequest{
			UserId:  input.UserID,
			Profile: toProtoProfile(&input.Profile),
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
