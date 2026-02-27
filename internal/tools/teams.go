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

type CreateTeamInput struct {
	Name        string `json:"name" jsonschema:"Team name (max 200 chars)"`
	Description string `json:"description,omitempty" jsonschema:"Optional description (max 1000 chars)"`
}

type GetTeamInput struct {
	TeamID string `json:"team_id" jsonschema:"Team UUID"`
}

type ListTeamsInput struct {
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type UpdateTeamInput struct {
	TeamID      string `json:"team_id" jsonschema:"Team UUID to update"`
	Name        string `json:"name,omitempty" jsonschema:"New team name"`
	Description string `json:"description,omitempty" jsonschema:"New description"`
}

type DeleteTeamInput struct {
	TeamID string `json:"team_id" jsonschema:"Team UUID to delete"`
}

type AddTeamMembersInput struct {
	TeamID  string   `json:"team_id" jsonschema:"Team UUID"`
	UserIDs []string `json:"user_ids" jsonschema:"User UUIDs to add (max 100)"`
}

type RemoveTeamMembersInput struct {
	TeamID  string   `json:"team_id" jsonschema:"Team UUID"`
	UserIDs []string `json:"user_ids" jsonschema:"User UUIDs to remove (max 100)"`
}

type ListTeamMembersInput struct {
	TeamID    string `json:"team_id" jsonschema:"Team UUID"`
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerTeamTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_team",
		Description: "Create a new organizational team (department/division).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateTeamInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Teams.CreateTeam(ctx, connect.NewRequest(&pidgrv1.CreateTeamRequest{
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
		Name:        "get_team",
		Description: "Retrieve a team by ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetTeamInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Teams.GetTeam(ctx, connect.NewRequest(&pidgrv1.GetTeamRequest{
			TeamId: input.TeamID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_teams",
		Description: "List teams in the organization with pagination.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTeamsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Teams.ListTeams(ctx, connect.NewRequest(&pidgrv1.ListTeamsRequest{
			Pagination: &pidgrv1.Pagination{
				PageSize:  clampPageSize(input.PageSize),
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
		Name:        "update_team",
		Description: "Update a team's name and/or description.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTeamInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Teams.UpdateTeam(ctx, connect.NewRequest(&pidgrv1.UpdateTeamRequest{
			TeamId:      input.TeamID,
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
		Name:        "delete_team",
		Description: "Delete a team and all its memberships. Default teams cannot be deleted.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteTeamInput) (*mcp.CallToolResult, any, error) {
		_, err := c.Teams.DeleteTeam(ctx, connect.NewRequest(&pidgrv1.DeleteTeamRequest{
			TeamId: input.TeamID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("Team deleted successfully"), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_team_members",
		Description: "Add users to a team (idempotent).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AddTeamMembersInput) (*mcp.CallToolResult, any, error) {
		if err := validateBatchSize(input.UserIDs, 100); err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		resp, err := c.Teams.AddTeamMembers(ctx, connect.NewRequest(&pidgrv1.AddTeamMembersRequest{
			TeamId:  input.TeamID,
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
		Name:        "remove_team_members",
		Description: "Remove users from a team (idempotent).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RemoveTeamMembersInput) (*mcp.CallToolResult, any, error) {
		if err := validateBatchSize(input.UserIDs, 100); err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		resp, err := c.Teams.RemoveTeamMembers(ctx, connect.NewRequest(&pidgrv1.RemoveTeamMembersRequest{
			TeamId:  input.TeamID,
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
		Name:        "list_team_members",
		Description: "List members of a team with pagination.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTeamMembersInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Teams.ListTeamMembers(ctx, connect.NewRequest(&pidgrv1.ListTeamMembersRequest{
			TeamId: input.TeamID,
			Pagination: &pidgrv1.Pagination{
				PageSize:  clampPageSize(input.PageSize),
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
}
