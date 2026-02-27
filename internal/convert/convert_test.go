// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package convert

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
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

func TestErrorResultConnectError(t *testing.T) {
	err := connect.NewError(connect.CodeNotFound, fmt.Errorf("campaign not found"))
	result, resultErr := ErrorResult(err)
	if resultErr != nil {
		t.Fatalf("unexpected error: %v", resultErr)
	}
	if !result.IsError {
		t.Fatal("expected IsError to be true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
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
