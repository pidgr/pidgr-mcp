// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package convert

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
)

func TestProtoResult(t *testing.T) {
	msg := &pidgrv1.GetCampaignResponse{
		Campaign: &pidgrv1.Campaign{
			Id:   "test-id",
			Name: "Test Campaign",
		},
	}
	result, err := ProtoResult(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.IsError {
		t.Fatal("expected IsError to be false")
	}
}

func TestProtoResultEmpty(t *testing.T) {
	msg := &pidgrv1.DeleteGroupResponse{}
	result, err := ProtoResult(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestErrorResultConnectNotFound(t *testing.T) {
	err := connect.NewError(connect.CodeNotFound, fmt.Errorf("campaign not found"))
	result, resultErr := ErrorResult(err)
	if resultErr != nil {
		t.Fatalf("unexpected error: %v", resultErr)
	}
	if !result.IsError {
		t.Fatal("expected IsError to be true")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "Not found" {
		t.Errorf("expected generic message %q, got %q", "Not found", text)
	}
}

func TestErrorResultConnectPermissionDenied(t *testing.T) {
	err := connect.NewError(connect.CodePermissionDenied, fmt.Errorf("user lacks CAMPAIGNS_WRITE"))
	result, resultErr := ErrorResult(err)
	if resultErr != nil {
		t.Fatalf("unexpected error: %v", resultErr)
	}
	if !result.IsError {
		t.Fatal("expected IsError to be true")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "Permission denied" {
		t.Errorf("expected generic message %q, got %q", "Permission denied", text)
	}
}

func TestErrorResultConnectInvalidArgument(t *testing.T) {
	err := connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name too long"))
	result, resultErr := ErrorResult(err)
	if resultErr != nil {
		t.Fatalf("unexpected error: %v", resultErr)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "Invalid input" {
		t.Errorf("expected generic message %q, got %q", "Invalid input", text)
	}
}

func TestErrorResultGenericError(t *testing.T) {
	err := fmt.Errorf("connection refused")
	result, resultErr := ErrorResult(err)
	if resultErr != nil {
		t.Fatalf("unexpected error: %v", resultErr)
	}
	if !result.IsError {
		t.Fatal("expected IsError to be true")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "Request failed" {
		t.Errorf("expected generic fallback %q, got %q", "Request failed", text)
	}
}

func TestErrorResultNotModified(t *testing.T) {
	err := connect.NewNotModifiedError(nil)
	result, resultErr := ErrorResult(err)
	if resultErr != nil {
		t.Fatalf("unexpected error: %v", resultErr)
	}
	if !result.IsError {
		t.Fatal("expected IsError to be true")
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if text != "Not modified" {
		t.Errorf("expected %q, got %q", "Not modified", text)
	}
}

func TestErrorResultDoesNotLeakDetails(t *testing.T) {
	// The backend error message should never appear in the result.
	backendMsg := "user lacks CAMPAIGNS_WRITE on org_abc123"
	err := connect.NewError(connect.CodePermissionDenied, fmt.Errorf("%s", backendMsg))
	result, _ := ErrorResult(err)
	text := result.Content[0].(*mcp.TextContent).Text
	if text == backendMsg || text == fmt.Sprintf("permission_denied: %s", backendMsg) {
		t.Errorf("error result leaked backend details: %q", text)
	}
}

func TestSuccessResult(t *testing.T) {
	result := SuccessResult("deleted successfully")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Fatal("expected IsError to be false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
}
