// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"github.com/pidgr/pidgr-mcp/internal/convert"
	"github.com/pidgr/pidgr-mcp/internal/transport"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── Input types ─────────────────────────────────────────────────────────────

type ListSessionRecordingsInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"Filter by campaign UUID"`
	DateFrom   string `json:"date_from,omitempty" jsonschema:"Start of time range (RFC 3339)"`
	DateTo     string `json:"date_to,omitempty" jsonschema:"End of time range (RFC 3339)"`
	PageSize   int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken  string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type GetSessionSnapshotsInput struct {
	RecordingID string `json:"recording_id" jsonschema:"PostHog recording ID"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerReplayTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_session_recordings",
		Description: "List session recordings with optional campaign and time range filters. Requires CAMPAIGNS_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListSessionRecordingsInput) (*mcp.CallToolResult, any, error) {
		protoReq := &pidgrv1.ListSessionRecordingsRequest{
			CampaignId: input.CampaignID,
			Pagination: &pidgrv1.Pagination{
				PageSize:  input.PageSize,
				PageToken: input.PageToken,
			},
		}

		if input.DateFrom != "" {
			if t, err := time.Parse(time.RFC3339, input.DateFrom); err == nil {
				protoReq.DateFrom = timestamppb.New(t)
			}
		}
		if input.DateTo != "" {
			if t, err := time.Parse(time.RFC3339, input.DateTo); err == nil {
				protoReq.DateTo = timestamppb.New(t)
			}
		}

		resp, err := c.Replays.ListSessionRecordings(ctx, connect.NewRequest(protoReq))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_session_snapshots",
		Description: "Fetch rrweb snapshot data for a session recording. Requires CAMPAIGNS_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetSessionSnapshotsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Replays.GetSessionSnapshots(ctx, connect.NewRequest(&pidgrv1.GetSessionSnapshotsRequest{
			RecordingId: input.RecordingID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
