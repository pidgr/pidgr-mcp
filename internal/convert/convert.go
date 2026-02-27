// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package convert

import (
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var marshaler = protojson.MarshalOptions{
	EmitUnpopulated: false,
}

// ProtoResult serializes a proto message to JSON and wraps it in an MCP CallToolResult.
func ProtoResult(msg proto.Message) (*mcp.CallToolResult, error) {
	data, err := marshaler.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal proto response: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil
}

// genericMessage maps Connect error codes to safe, user-facing messages.
var genericMessage = map[connect.Code]string{
	connect.CodeCanceled:           "Request canceled",
	connect.CodeInvalidArgument:    "Invalid input",
	connect.CodeNotFound:           "Not found",
	connect.CodeAlreadyExists:      "Already exists",
	connect.CodePermissionDenied:   "Permission denied",
	connect.CodeResourceExhausted:  "Too many requests",
	connect.CodeFailedPrecondition: "Operation not allowed in current state",
	connect.CodeAborted:            "Operation aborted",
	connect.CodeOutOfRange:         "Value out of range",
	connect.CodeUnimplemented:      "Not supported",
	connect.CodeInternal:           "Internal error",
	connect.CodeUnavailable:        "Service unavailable",
	connect.CodeDataLoss:           "Data loss",
	connect.CodeUnauthenticated:    "Authentication required",
	connect.CodeDeadlineExceeded:   "Request timed out",
}

// ErrorResult converts an error into an MCP error result with sanitized messages.
func ErrorResult(err error) (*mcp.CallToolResult, error) {
	if connect.IsNotModifiedError(err) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Not modified"},
			},
		}, nil
	}

	if code := connect.CodeOf(err); code != connect.CodeUnknown {
		slog.Warn("backend error", "code", code, "detail", err.Error())
		msg := "Request failed"
		if m, ok := genericMessage[code]; ok {
			msg = m
		}
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: msg},
			},
		}, nil
	}

	slog.Warn("unexpected error", "detail", err.Error())
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Request failed"},
		},
	}, nil
}

// SuccessResult returns a simple success message for void responses.
func SuccessResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}
