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

type QueryHeatmapDataInput struct {
	ScreenName     string   `json:"screen_name" jsonschema:"Screen name from React Navigation route"`
	DateFrom       string   `json:"date_from,omitempty" jsonschema:"Start of time range (RFC 3339)"`
	DateTo         string   `json:"date_to,omitempty" jsonschema:"End of time range (RFC 3339)"`
	CampaignID     string   `json:"campaign_id,omitempty" jsonschema:"Filter by campaign UUID"`
	UserID         string   `json:"user_id,omitempty" jsonschema:"Filter by user UUID (required for USER_SPECIFIC mode)"`
	GridResolution float32  `json:"grid_resolution,omitempty" jsonschema:"Grid resolution (0.005 to 0.1, default 0.02)"`
	Mode           string   `json:"mode,omitempty" jsonschema:"Aggregation mode: TOTAL (default), MEDIAN, or USER_SPECIFIC"`
	EventTypes     []string `json:"event_types,omitempty" jsonschema:"Filter by event types: TAP, LONG_PRESS, SCROLL, ACTION_CLICK"`
}

type ListScreenshotsInput struct{}

// ── Registration ────────────────────────────────────────────────────────────

func registerHeatmapTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "query_heatmap_data",
		Description: "Query aggregated touch data for heatmap rendering. Requires CAMPAIGNS_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input QueryHeatmapDataInput) (*mcp.CallToolResult, any, error) {
		protoReq := &pidgrv1.QueryHeatmapDataRequest{
			ScreenName:     input.ScreenName,
			CampaignId:     input.CampaignID,
			UserId:         input.UserID,
			GridResolution: input.GridResolution,
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

		if input.Mode != "" {
			if v, ok := pidgrv1.HeatmapMode_value[input.Mode]; ok {
				protoReq.Mode = pidgrv1.HeatmapMode(v)
			} else if v, ok := pidgrv1.HeatmapMode_value["HEATMAP_MODE_"+input.Mode]; ok {
				protoReq.Mode = pidgrv1.HeatmapMode(v)
			}
		}

		for _, et := range input.EventTypes {
			if v, ok := pidgrv1.TouchEventType_value[et]; ok {
				protoReq.EventTypes = append(protoReq.EventTypes, pidgrv1.TouchEventType(v))
			} else if v, ok := pidgrv1.TouchEventType_value["TOUCH_EVENT_TYPE_"+et]; ok {
				protoReq.EventTypes = append(protoReq.EventTypes, pidgrv1.TouchEventType(v))
			}
		}

		resp, err := c.Heatmaps.QueryHeatmapData(ctx, connect.NewRequest(protoReq))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_screenshots",
		Description: "List available screen screenshots for heatmap backgrounds. Requires CAMPAIGNS_READ permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListScreenshotsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Heatmaps.ListScreenshots(ctx, connect.NewRequest(&pidgrv1.ListScreenshotsRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
