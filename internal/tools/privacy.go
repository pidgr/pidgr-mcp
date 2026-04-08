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

type ExportUserDataInput struct {
	UserID string `json:"user_id" jsonschema:"User email or UUID to export data for"`
}

type DeleteUserDataInput struct {
	UserID    string `json:"user_id" jsonschema:"User email or UUID to delete data for"`
	Anonymize bool   `json:"anonymize" jsonschema:"If true, anonymize instead of hard-delete"`
}

type CancelDeletionInput struct {
	RequestID    string `json:"request_id" jsonschema:"Privacy request UUID to cancel"`
	ConfirmationEmail string `json:"confirm_email" jsonschema:"Target user email (confirmation)"`
}

type ImmediateDeleteInput struct {
	RequestID    string `json:"request_id" jsonschema:"Privacy request UUID to delete immediately"`
	ConfirmationEmail string `json:"confirm_email" jsonschema:"Target user email (confirmation)"`
}

type ListPrivacyRequestsInput struct {
	RequestType string `json:"request_type,omitempty" jsonschema:"Filter by type: export, delete, rectify, restrict"`
}

type RectifyUserDataInput struct {
	UserID string `json:"user_id" jsonschema:"User email or UUID to rectify data for"`
}

type RestrictProcessingInput struct {
	UserID string `json:"user_id" jsonschema:"User email or UUID to restrict processing for"`
}

type GetDataExistenceConfirmationInput struct {
	UserID string `json:"user_id" jsonschema:"User email or UUID to check data existence for"`
}

type ListMyPrivacyRequestsInput struct {
	PageSize    int32  `json:"page_size,omitempty" jsonschema:"Max items per page (1-100, default 25)"`
	PageToken   string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
	RequestType string `json:"request_type,omitempty" jsonschema:"Filter by type: export, rectify (empty = all)"`
	Status      string `json:"status,omitempty" jsonschema:"Filter by status: PENDING, COMPLETED, CANCELLED (empty = all)"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerPrivacyTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "export_user_data",
		Description: "Export all personal data for a user as a downloadable ZIP (GDPR Art. 15).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ExportUserDataInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Privacy.ExportUserData(ctx, connect.NewRequest(&pidgrv1.ExportUserDataRequest{
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
		Name:        "delete_user_data",
		Description: "Schedule deletion or anonymization of all personal data for a user (GDPR Art. 17). Includes 30-day grace period.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteUserDataInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Privacy.DeleteUserData(ctx, connect.NewRequest(&pidgrv1.DeleteUserDataRequest{
			UserId:    input.UserID,
			Anonymize: input.Anonymize,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "cancel_deletion",
		Description: "Cancel a pending data deletion request before the grace period expires.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CancelDeletionInput) (*mcp.CallToolResult, any, error) {
		_, err := c.Privacy.CancelDeletion(ctx, connect.NewRequest(&pidgrv1.CancelDeletionRequest{
			RequestId:    input.RequestID,
			ConfirmationEmail: input.ConfirmationEmail,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("Deletion cancelled successfully"), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "immediate_delete",
		Description: "Skip the grace period and delete user data immediately.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ImmediateDeleteInput) (*mcp.CallToolResult, any, error) {
		_, err := c.Privacy.ImmediateDelete(ctx, connect.NewRequest(&pidgrv1.ImmediateDeleteRequest{
			RequestId:    input.RequestID,
			ConfirmationEmail: input.ConfirmationEmail,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		return convert.SuccessResult("User data deleted immediately"), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_privacy_requests",
		Description: "List privacy requests (exports, deletions, rectifications) for the organization.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListPrivacyRequestsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Privacy.ListPrivacyRequests(ctx, connect.NewRequest(&pidgrv1.ListPrivacyRequestsRequest{
			RequestType: input.RequestType,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rectify_user_data",
		Description: "Rectify personal data for a user (GDPR Art. 16).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RectifyUserDataInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Privacy.RectifyUserData(ctx, connect.NewRequest(&pidgrv1.RectifyUserDataRequest{
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
		Name:        "restrict_processing",
		Description: "Restrict processing of personal data for a user (GDPR Art. 18).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RestrictProcessingInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Privacy.RestrictProcessing(ctx, connect.NewRequest(&pidgrv1.RestrictProcessingRequest{
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
		Name:        "get_data_existence_confirmation",
		Description: "Confirm whether personal data exists for a user (LGPD confirmação de existência).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetDataExistenceConfirmationInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Privacy.GetDataExistenceConfirmation(ctx, connect.NewRequest(&pidgrv1.GetDataExistenceConfirmationRequest{
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
		Name:        "list_my_privacy_requests",
		Description: "List the authenticated user's own privacy requests (exports, rectifications). No admin permission required.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListMyPrivacyRequestsInput) (*mcp.CallToolResult, any, error) {
		status := pidgrv1.PrivacyRequestStatus_PRIVACY_REQUEST_STATUS_UNSPECIFIED
		if v, ok := pidgrv1.PrivacyRequestStatus_value[input.Status]; ok {
			status = pidgrv1.PrivacyRequestStatus(v)
		} else if v, ok := pidgrv1.PrivacyRequestStatus_value["PRIVACY_REQUEST_STATUS_"+input.Status]; ok {
			status = pidgrv1.PrivacyRequestStatus(v)
		}
		resp, err := c.Privacy.ListMyPrivacyRequests(ctx, connect.NewRequest(&pidgrv1.ListMyPrivacyRequestsRequest{
			PageSize:    clampPageSize(input.PageSize),
			PageToken:   input.PageToken,
			RequestType: input.RequestType,
			Status:      status,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
