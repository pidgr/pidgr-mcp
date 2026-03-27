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

type ListAuditEventsInput struct {
	EventType int32  `json:"event_type,omitempty" jsonschema:"Filter by event type (numeric enum value, 0 = all)"`
	StartTime string `json:"start_time,omitempty" jsonschema:"Events after this time (RFC 3339)"`
	EndTime   string `json:"end_time,omitempty" jsonschema:"Events before this time (RFC 3339)"`
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max events to return (default 50, max 100)"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
}

type ExportAuditTrailInput struct {
	Format    string `json:"format" jsonschema:"Export format: csv or json"`
	StartTime string `json:"start_time,omitempty" jsonschema:"Export events after this time (RFC 3339)"`
	EndTime   string `json:"end_time,omitempty" jsonschema:"Export events before this time (RFC 3339)"`
}

type ListAuditExportsInput struct{}

// ── Registration ────────────────────────────────────────────────────────────

func registerAuditTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_audit_events",
		Description: "List audit trail events with optional filters by type, date range, and pagination.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListAuditEventsInput) (*mcp.CallToolResult, any, error) {
		r := &pidgrv1.ListAuditEventsRequest{
			EventType: pidgrv1.AuditEventType(input.EventType),
			PageSize:  input.PageSize,
			PageToken: input.PageToken,
		}
		if input.StartTime != "" {
			if t, err := time.Parse(time.RFC3339, input.StartTime); err == nil {
				r.StartTime = timestamppb.New(t)
			}
		}
		if input.EndTime != "" {
			if t, err := time.Parse(time.RFC3339, input.EndTime); err == nil {
				r.EndTime = timestamppb.New(t)
			}
		}
		resp, err := c.Audit.ListAuditEvents(ctx, connect.NewRequest(r))
		if err != nil {
			res, _ := convert.ErrorResult(err)
			return res, nil, nil
		}
		res, err := convert.ProtoResult(resp.Msg)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "export_audit_trail",
		Description: "Export the audit trail to S3 as CSV or JSON. Returns a pre-signed download URL.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ExportAuditTrailInput) (*mcp.CallToolResult, any, error) {
		r := &pidgrv1.ExportAuditTrailRequest{}
		switch input.Format {
		case "csv":
			r.Format = pidgrv1.AuditExportFormat_AUDIT_EXPORT_FORMAT_CSV
		default:
			r.Format = pidgrv1.AuditExportFormat_AUDIT_EXPORT_FORMAT_JSON
		}
		if input.StartTime != "" {
			if t, err := time.Parse(time.RFC3339, input.StartTime); err == nil {
				r.StartTime = timestamppb.New(t)
			}
		}
		if input.EndTime != "" {
			if t, err := time.Parse(time.RFC3339, input.EndTime); err == nil {
				r.EndTime = timestamppb.New(t)
			}
		}
		resp, err := c.Audit.ExportAuditTrail(ctx, connect.NewRequest(r))
		if err != nil {
			res, _ := convert.ErrorResult(err)
			return res, nil, nil
		}
		res, err := convert.ProtoResult(resp.Msg)
		return res, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_audit_exports",
		Description: "List audit export history for the organization.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListAuditExportsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Audit.ListAuditExports(ctx, connect.NewRequest(&pidgrv1.ListAuditExportsRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
