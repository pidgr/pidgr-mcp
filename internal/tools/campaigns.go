// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pidgr/pidgr-mcp/internal/convert"
	"github.com/pidgr/pidgr-mcp/internal/transport"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
)

// ── Input types ─────────────────────────────────────────────────────────────

type CreateCampaignInput struct {
	Name            string                 `json:"name" jsonschema:"Campaign name (max 200 chars)"`
	TemplateID      string                 `json:"template_id" jsonschema:"Template UUID to use for rendering"`
	TemplateVersion int32                  `json:"template_version,omitempty" jsonschema:"Template version to pin"`
	UserIDs         []string               `json:"user_ids,omitempty" jsonschema:"Audience user IDs (max 100000)"`
	SenderName      string                 `json:"sender_name" jsonschema:"Sender shown to recipients. MUST exactly match the name of a team registered in the org — the server rejects anything else (INVALID_ARGUMENT). Call list_teams and pick an existing team name; never invent or guess one. If no suitable team exists, create it with create_team first."`
	Title           string                 `json:"title,omitempty" jsonschema:"Optional user-facing title override (max 200 chars)"`
	Workflow        json.RawMessage        `json:"workflow" jsonschema:"REQUIRED protojson WorkflowDefinition. Use full STEP_TYPE_* enum names and the typed oneof config key (sendNotification/deadlineCheck/...). Example: {\"steps\":[{\"id\":\"send\",\"type\":\"STEP_TYPE_SEND_NOTIFICATION\",\"sendNotification\":{\"type\":\"push\",\"actionType\":\"ACTION_TYPE_ACK\"},\"transitions\":{\"completed\":\"wait\"}},{\"id\":\"wait\",\"type\":\"STEP_TYPE_DEADLINE_CHECK\",\"deadlineCheck\":{\"delay\":\"5m\"},\"transitions\":{\"completed\":\"done\"}},{\"id\":\"done\",\"type\":\"STEP_TYPE_MARK_MISSED\"}]}. Or copy the workflow from an existing campaign via list_campaigns/get_campaign."`
	Audience        []*AudienceMemberInput `json:"audience,omitempty" jsonschema:"Rich audience with per-user template variables"`
}

type AudienceMemberInput struct {
	UserID    string            `json:"user_id" jsonschema:"User UUID"`
	Variables map[string]string `json:"variables,omitempty" jsonschema:"Template variable values for this user"`
}

type UpdateCampaignInput struct {
	CampaignID      string                 `json:"campaign_id" jsonschema:"Campaign UUID to update"`
	Name            string                 `json:"name,omitempty" jsonschema:"Updated campaign name"`
	SenderName      string                 `json:"sender_name,omitempty" jsonschema:"Updated sender. MUST exactly match a registered team name (see list_teams) — the server rejects anything else. Omit to keep the current sender."`
	Title           string                 `json:"title,omitempty" jsonschema:"Updated title override"`
	TemplateID      string                 `json:"template_id,omitempty" jsonschema:"Updated template UUID"`
	TemplateVersion int32                  `json:"template_version,omitempty" jsonschema:"Updated template version"`
	Workflow        json.RawMessage        `json:"workflow,omitempty" jsonschema:"Updated workflow DAG"`
	Audience        []*AudienceMemberInput `json:"audience,omitempty" jsonschema:"Replaces the campaign's frozen audience WHOLESALE with this list (only valid while the campaign is CREATED). Omit to leave the audience unchanged; an explicit empty array clears it. Read the current audience first with get_campaign_audience and send the complete new list — partial additions are not merged."`
}

type GetCampaignAudienceInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"Campaign UUID whose frozen audience to read"`
}

type StartCampaignInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"Campaign UUID to start"`
}

type GetCampaignInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"Campaign UUID to retrieve"`
}

type ListCampaignsInput struct {
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type CancelCampaignInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"Campaign UUID to cancel"`
}

type ListDeliveriesInput struct {
	CampaignID   string `json:"campaign_id" jsonschema:"Campaign UUID"`
	StatusFilter string `json:"status_filter,omitempty" jsonschema:"Filter by delivery status (PENDING/SENT/DELIVERED/ACKNOWLEDGED/MISSED/NO_DEVICE/FAILED)"`
	PageSize     int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken    string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type GetCampaignArchetypeBreakdownInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"Campaign UUID to compute the archetype-share breakdown for"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerCampaignTools(s *mcp.Server, c *transport.Clients) {
	registerCampaignArchetypeBreakdownTool(s, c)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_campaign",
		Description: "Create a new campaign with a template, audience, and workflow. The workflow is REQUIRED — the server rejects a campaign with no workflow (INVALID_ARGUMENT) and does not substitute a default. Pass a protojson WorkflowDefinition in the `workflow` field (see its schema for the format/example), or copy one from an existing campaign via list_campaigns/get_campaign. `sender_name` MUST be the name of a registered team (call list_teams first; the server rejects anything else). Use list_templates to find template UUIDs, and list_users or list_team_members/list_group_members to resolve audience user UUIDs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateCampaignInput) (*mcp.CallToolResult, any, error) {
		if err := validateBatchSize(input.UserIDs, maxBatchSize); err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		if len(input.Workflow) == 0 {
			r, _ := convert.ErrorResult(errors.New("workflow is required: pass a protojson WorkflowDefinition (e.g. a STEP_TYPE_SEND_NOTIFICATION → STEP_TYPE_DEADLINE_CHECK → STEP_TYPE_MARK_MISSED DAG), or copy the workflow from an existing campaign via list_campaigns/get_campaign"))
			return r, nil, nil
		}
		wf, err := workflowFromJSON(input.Workflow)
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		var audience []*pidgrv1.AudienceMember
		for _, a := range input.Audience {
			audience = append(audience, &pidgrv1.AudienceMember{
				UserId:    a.UserID,
				Variables: a.Variables,
			})
		}
		resp, err := c.Campaigns.CreateCampaign(ctx, connect.NewRequest(&pidgrv1.CreateCampaignRequest{
			Name:            input.Name,
			TemplateId:      input.TemplateID,
			TemplateVersion: input.TemplateVersion,
			UserIds:         input.UserIDs,
			Workflow:        wf,
			SenderName:      input.SenderName,
			Title:           input.Title,
			Audience:        audience,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_campaign",
		Description: "Update a draft campaign (CREATED status only — started campaigns can only be duplicated). Only non-empty fields are changed. A new `sender_name` MUST be a registered team name (see list_teams). Providing `audience` replaces the frozen audience wholesale. Use list_campaigns to find the campaign UUID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateCampaignInput) (*mcp.CallToolResult, any, error) {
		wf, err := workflowFromJSON(input.Workflow)
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		// nil = field omitted = no audience change. A non-nil (possibly empty)
		// list is a wholesale replacement — the wrapper message carries that
		// presence distinction on the wire.
		var replacement *pidgrv1.AudienceReplacement
		if input.Audience != nil {
			members := make([]*pidgrv1.AudienceMember, len(input.Audience))
			for i, m := range input.Audience {
				members[i] = &pidgrv1.AudienceMember{UserId: m.UserID, Variables: m.Variables}
			}
			replacement = &pidgrv1.AudienceReplacement{Members: members}
		}
		resp, err := c.Campaigns.UpdateCampaign(ctx, connect.NewRequest(&pidgrv1.UpdateCampaignRequest{
			CampaignId:          input.CampaignID,
			Name:                input.Name,
			SenderName:          input.SenderName,
			Title:               input.Title,
			TemplateId:          input.TemplateID,
			TemplateVersion:     input.TemplateVersion,
			Workflow:            wf,
			AudienceReplacement: replacement,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_campaign_audience",
		Description: "Read a campaign's frozen audience snapshot, enriched per entry with email, display name, and whether the member is still reachable (active). Use before update_campaign's `audience` to build the complete replacement list. Empty for campaigns without a snapshot.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetCampaignAudienceInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Campaigns.GetCampaignAudience(ctx, connect.NewRequest(&pidgrv1.GetCampaignAudienceRequest{
			CampaignId: input.CampaignID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_campaign",
		Description: "Start a campaign's workflow execution. Use list_campaigns to find the campaign UUID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input StartCampaignInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Campaigns.StartCampaign(ctx, connect.NewRequest(&pidgrv1.StartCampaignRequest{
			CampaignId: input.CampaignID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_campaign",
		Description: "Retrieve a single campaign by UUID. Use list_campaigns to find available campaign UUIDs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetCampaignInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Campaigns.GetCampaign(ctx, connect.NewRequest(&pidgrv1.GetCampaignRequest{
			CampaignId: input.CampaignID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_campaigns",
		Description: "List campaigns for the organization with pagination. Call this first to discover campaign UUIDs before using other campaign tools.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListCampaignsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Campaigns.ListCampaigns(ctx, connect.NewRequest(&pidgrv1.ListCampaignsRequest{
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
		Name:        "cancel_campaign",
		Description: "Cancel a running campaign. Use list_campaigns to find the campaign UUID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CancelCampaignInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Campaigns.CancelCampaign(ctx, connect.NewRequest(&pidgrv1.CancelCampaignRequest{
			CampaignId: input.CampaignID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_deliveries",
		Description: "List delivery records for a campaign, optionally filtered by status. Use list_campaigns to find the campaign UUID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListDeliveriesInput) (*mcp.CallToolResult, any, error) {
		statusFilter := pidgrv1.DeliveryStatus_DELIVERY_STATUS_UNSPECIFIED
		if input.StatusFilter != "" {
			if v, ok := pidgrv1.DeliveryStatus_value[input.StatusFilter]; ok {
				statusFilter = pidgrv1.DeliveryStatus(v)
			} else if v, ok := pidgrv1.DeliveryStatus_value["DELIVERY_STATUS_"+input.StatusFilter]; ok {
				statusFilter = pidgrv1.DeliveryStatus(v)
			}
		}
		resp, err := c.Campaigns.ListDeliveries(ctx, connect.NewRequest(&pidgrv1.ListDeliveriesRequest{
			CampaignId:   input.CampaignID,
			StatusFilter: statusFilter,
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

// registerCampaignArchetypeBreakdownTool exposes
// pidgr.v1.CampaignService.GetCampaignArchetypeBreakdown. The proto response
// carries the email-engagement fields (`email_delivered_count`,
// `email_open_rate_real`, `email_open_rate_raw` per ArchetypeShareShift); they
// pass through naturally via protojson.Marshal on the generated proto type.
//
// Factored out of registerCampaignTools so the handler can be exercised in
// isolation in integrations_test.go without spinning up the full campaign tool
// surface.
func registerCampaignArchetypeBreakdownTool(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_campaign_archetype_breakdown",
		Description: "Get the per-archetype tendency-shift breakdown for a campaign. " +
			"Returns shifts, before_snapshot_at, after_snapshot_at, insufficient_history. " +
			"Each shift includes the email-engagement fields when populated: " +
			"`email_delivered_count`, `email_open_rate_real`, `email_open_rate_raw`. " +
			"Returns `shifts: []` with `insufficient_history: true` when the group has fewer than two clustering snapshots — NOT a NOT_FOUND.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input GetCampaignArchetypeBreakdownInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Campaigns.GetCampaignArchetypeBreakdown(ctx, connect.NewRequest(&pidgrv1.GetCampaignArchetypeBreakdownRequest{
			CampaignId: input.CampaignID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
