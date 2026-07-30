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

type GetGroupArchetypesInput struct {
	GroupID string `json:"group_id" jsonschema:"Group UUID to query archetypes for"`
}

type PredictCampaignACKInput struct {
	GroupID           string `json:"group_id" jsonschema:"Target audience group UUID"`
	TemplateType      string `json:"template_type,omitempty" jsonschema:"Template type for prediction refinement"`
	WorkflowStepCount int32  `json:"workflow_step_count,omitempty" jsonschema:"Number of workflow steps"`
}

type GetCampaignAdvisoryInput struct {
	GroupID           string `json:"group_id" jsonschema:"Target audience group UUID"`
	TemplateID        string `json:"template_id,omitempty" jsonschema:"Template UUID for advisory context"`
	TemplateVersion   int32  `json:"template_version,omitempty" jsonschema:"Template version"`
	WorkflowStepCount int32  `json:"workflow_step_count,omitempty" jsonschema:"Number of workflow steps"`
}

type GetInsightNarrativeInput struct {
	GroupID    string `json:"group_id" jsonschema:"Group UUID to generate a narrative for"`
	PromptName string `json:"prompt_name" jsonschema:"Prompt template name (e.g., 'campaign-advisory', 'archetype-explanation')"`
}

type TriggerMLPipelineInput struct{}

type TriggerArchetypeClusteringInput struct {
	GroupID string `json:"group_id" jsonschema:"Group UUID to recluster. The org comes from the caller's token"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerInsightsTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_group_archetypes",
		Description: "Get behavioral archetypes for a group based on anonymous campaign interaction patterns. Returns cohort-level patterns (e.g., 'Swift Acknowledger', 'Thorough Reader'). Empty if insufficient data (<50 campaigns).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetGroupArchetypesInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Insights.GetGroupArchetypes(ctx, connect.NewRequest(&pidgrv1.GetGroupArchetypesRequest{
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
		Name:        "predict_campaign_ack",
		Description: "Predict the cohort-level ACK rate for a campaign targeting a specific group. Returns a confidence interval that narrows as more campaign data accumulates. Never predicts individual behavior.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PredictCampaignACKInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Insights.PredictCampaignACK(ctx, connect.NewRequest(&pidgrv1.PredictCampaignACKRequest{
			GroupId:           input.GroupID,
			TemplateType:      input.TemplateType,
			WorkflowStepCount: input.WorkflowStepCount,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_campaign_advisory",
		Description: "Get campaign configuration advisory: predicted ACK rate + behavioral archetypes + suggested escalation delay. Informational only — never drives automated decisions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetCampaignAdvisoryInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Insights.GetCampaignAdvisory(ctx, connect.NewRequest(&pidgrv1.GetCampaignAdvisoryRequest{
			GroupId:           input.GroupID,
			TemplateId:        input.TemplateID,
			TemplateVersion:   input.TemplateVersion,
			WorkflowStepCount: input.WorkflowStepCount,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_insight_narrative",
		Description: "Generate an AI-powered narrative summary of a group's insights. Combines archetype, prediction, and campaign data into human-readable analysis using the specified prompt template.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetInsightNarrativeInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Insights.GetInsightNarrative(ctx, connect.NewRequest(&pidgrv1.GetInsightNarrativeRequest{
			GroupId:    input.GroupID,
			PromptName: input.PromptName,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "trigger_ml_pipeline",
		Description: "Manually trigger the ML training pipeline for the organization. Rate-limited by the org's monthly quota (default 3/month, auto-resets). Returns remaining retrains this month and the last training timestamp.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ TriggerMLPipelineInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Insights.TriggerMLPipeline(ctx, connect.NewRequest(&pidgrv1.TriggerMLPipelineRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "trigger_archetype_clustering",
		Description: "Recluster archetypes for a SINGLE group, reusing the already-deployed clustering model — no training runs, so it is much cheaper than trigger_ml_pipeline. " +
			"Both draw on the same monthly manual-retrain quota, so prefer this one when only one group's archetypes are stale.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input TriggerArchetypeClusteringInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Insights.TriggerArchetypeClustering(ctx, connect.NewRequest(&pidgrv1.TriggerArchetypeClusteringRequest{
			GroupId: input.GroupID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
