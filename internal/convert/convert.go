// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package convert

import (
	"fmt"

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

// ErrorResult converts a Connect/gRPC error into an MCP error result.
func ErrorResult(err error) (*mcp.CallToolResult, error) {
	if connectErr := new(connect.Error); connect.IsNotModifiedError(err) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "not modified"},
			},
		}, nil
	} else if ok := connect.CodeOf(err); ok != connect.CodeUnknown {
		_ = connectErr
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("%s: %s", ok, err.Error())},
			},
		}, nil
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: err.Error()},
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
