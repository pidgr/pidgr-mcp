// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pidgr/pidgr-mcp/internal/transport"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"github.com/pidgr/pidgr-proto/gen/go/pidgr/v1/pidgrv1connect"
)

// ─── Schema registration tests ──────────────────────────────────────────────

func TestIntegrationsToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{
		"list_reachabilities_for_user",
		"upsert_reachability",
		"remove_reachability",
		"get_cost_cap_policy",
		"set_cost_cap_policy",
		"get_campaign_archetype_breakdown",
	}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("integrations tool %q not registered", name)
		}
	}
}

func TestListReachabilitiesForUserSchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "list_reachabilities_for_user")
	if tool == nil {
		t.Fatal("list_reachabilities_for_user not registered")
	}
	if tool.Description == "" {
		t.Error("list_reachabilities_for_user has empty description")
	}
	if !schemaHasProperty(t, tool, "org_id") {
		t.Error("list_reachabilities_for_user missing org_id property")
	}
	if !schemaHasProperty(t, tool, "user_id") {
		t.Error("list_reachabilities_for_user missing user_id property")
	}
}

func TestUpsertReachabilitySchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "upsert_reachability")
	if tool == nil {
		t.Fatal("upsert_reachability not registered")
	}
	if tool.Description == "" {
		t.Error("upsert_reachability has empty description")
	}
	// Sensitivity advisory must appear in the description.
	if !strings.Contains(strings.ToLower(tool.Description), "sensitive") {
		t.Errorf("upsert_reachability description must mention that identifier_plaintext is sensitive, got: %s", tool.Description)
	}
	for _, prop := range []string{"org_id", "user_id", "channel", "identifier_plaintext"} {
		if !schemaHasProperty(t, tool, prop) {
			t.Errorf("upsert_reachability missing %s property", prop)
		}
	}
}

func TestRemoveReachabilitySchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "remove_reachability")
	if tool == nil {
		t.Fatal("remove_reachability not registered")
	}
	for _, prop := range []string{"org_id", "user_id", "channel"} {
		if !schemaHasProperty(t, tool, prop) {
			t.Errorf("remove_reachability missing %s property", prop)
		}
	}
}

func TestGetCostCapPolicySchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "get_cost_cap_policy")
	if tool == nil {
		t.Fatal("get_cost_cap_policy not registered")
	}
	for _, prop := range []string{"org_id", "channel"} {
		if !schemaHasProperty(t, tool, prop) {
			t.Errorf("get_cost_cap_policy missing %s property", prop)
		}
	}
}

func TestSetCostCapPolicySchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "set_cost_cap_policy")
	if tool == nil {
		t.Fatal("set_cost_cap_policy not registered")
	}
	for _, prop := range []string{"org_id", "channel", "cap_micros"} {
		if !schemaHasProperty(t, tool, prop) {
			t.Errorf("set_cost_cap_policy missing %s property", prop)
		}
	}
}

func TestGetCampaignArchetypeBreakdownSchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "get_campaign_archetype_breakdown")
	if tool == nil {
		t.Fatal("get_campaign_archetype_breakdown not registered")
	}
	if !schemaHasProperty(t, tool, "campaign_id") {
		t.Error("get_campaign_archetype_breakdown missing campaign_id property")
	}
	// Description should call out the email-open-rate fields so agents
	// know they can rely on this tool to surface those metrics.
	if !strings.Contains(strings.ToLower(tool.Description), "email") {
		t.Errorf("get_campaign_archetype_breakdown description should mention email-engagement fields, got: %s", tool.Description)
	}
}

// ─── Behavior tests with a mock IntegrationsService client ──────────────────

// fakeIntegrationsClient is a hand-rolled stub of pidgrv1connect.IntegrationsServiceClient
// used to exercise the tool handlers in isolation from the real Connect transport.
type fakeIntegrationsClient struct {
	listReachabilityForUserResp *pidgrv1.ListReachabilityForUserResponse
	listReachabilityForUserErr  error
	listReachabilityForUserReq  *pidgrv1.ListReachabilityForUserRequest

	upsertReachabilityResp *pidgrv1.UpsertReachabilityResponse
	upsertReachabilityErr  error
	upsertReachabilityReq  *pidgrv1.UpsertReachabilityRequest

	removeReachabilityResp *pidgrv1.RemoveReachabilityResponse
	removeReachabilityErr  error
	removeReachabilityReq  *pidgrv1.RemoveReachabilityRequest

	getCostCapPolicyResp *pidgrv1.GetCostCapPolicyResponse
	getCostCapPolicyErr  error
	getCostCapPolicyReq  *pidgrv1.GetCostCapPolicyRequest

	setCostCapPolicyResp *pidgrv1.SetCostCapPolicyResponse
	setCostCapPolicyErr  error
	setCostCapPolicyReq  *pidgrv1.SetCostCapPolicyRequest
}

var _ pidgrv1connect.IntegrationsServiceClient = (*fakeIntegrationsClient)(nil)

func (f *fakeIntegrationsClient) DispatchToChannel(_ context.Context, _ *connect.Request[pidgrv1.DispatchToChannelRequest]) (*connect.Response[pidgrv1.DispatchToChannelResponse], error) {
	return nil, errors.New("not used")
}

func (f *fakeIntegrationsClient) CreateChannelConnectLink(_ context.Context, _ *connect.Request[pidgrv1.CreateChannelConnectLinkRequest]) (*connect.Response[pidgrv1.CreateChannelConnectLinkResponse], error) {
	return nil, errors.New("not used")
}

func (f *fakeIntegrationsClient) UpsertReachability(_ context.Context, req *connect.Request[pidgrv1.UpsertReachabilityRequest]) (*connect.Response[pidgrv1.UpsertReachabilityResponse], error) {
	f.upsertReachabilityReq = req.Msg
	if f.upsertReachabilityErr != nil {
		return nil, f.upsertReachabilityErr
	}
	return connect.NewResponse(f.upsertReachabilityResp), nil
}

func (f *fakeIntegrationsClient) RemoveReachability(_ context.Context, req *connect.Request[pidgrv1.RemoveReachabilityRequest]) (*connect.Response[pidgrv1.RemoveReachabilityResponse], error) {
	f.removeReachabilityReq = req.Msg
	if f.removeReachabilityErr != nil {
		return nil, f.removeReachabilityErr
	}
	return connect.NewResponse(f.removeReachabilityResp), nil
}

func (f *fakeIntegrationsClient) GetReachability(_ context.Context, _ *connect.Request[pidgrv1.GetReachabilityRequest]) (*connect.Response[pidgrv1.GetReachabilityResponse], error) {
	return nil, errors.New("not used")
}

func (f *fakeIntegrationsClient) ListReachabilityForUser(_ context.Context, req *connect.Request[pidgrv1.ListReachabilityForUserRequest]) (*connect.Response[pidgrv1.ListReachabilityForUserResponse], error) {
	f.listReachabilityForUserReq = req.Msg
	if f.listReachabilityForUserErr != nil {
		return nil, f.listReachabilityForUserErr
	}
	return connect.NewResponse(f.listReachabilityForUserResp), nil
}

func (f *fakeIntegrationsClient) GetRegionPolicy(_ context.Context, _ *connect.Request[pidgrv1.GetRegionPolicyRequest]) (*connect.Response[pidgrv1.GetRegionPolicyResponse], error) {
	return nil, errors.New("not used")
}

func (f *fakeIntegrationsClient) SetRegionPolicy(_ context.Context, _ *connect.Request[pidgrv1.SetRegionPolicyRequest]) (*connect.Response[pidgrv1.SetRegionPolicyResponse], error) {
	return nil, errors.New("not used")
}

func (f *fakeIntegrationsClient) GetOrgWebhookConfig(_ context.Context, _ *connect.Request[pidgrv1.GetOrgWebhookConfigRequest]) (*connect.Response[pidgrv1.GetOrgWebhookConfigResponse], error) {
	return connect.NewResponse(&pidgrv1.GetOrgWebhookConfigResponse{}), nil
}

func (f *fakeIntegrationsClient) SetOrgWebhookConfig(_ context.Context, _ *connect.Request[pidgrv1.SetOrgWebhookConfigRequest]) (*connect.Response[pidgrv1.SetOrgWebhookConfigResponse], error) {
	return connect.NewResponse(&pidgrv1.SetOrgWebhookConfigResponse{}), nil
}

func (f *fakeIntegrationsClient) GetCostCapPolicy(_ context.Context, req *connect.Request[pidgrv1.GetCostCapPolicyRequest]) (*connect.Response[pidgrv1.GetCostCapPolicyResponse], error) {
	f.getCostCapPolicyReq = req.Msg
	if f.getCostCapPolicyErr != nil {
		return nil, f.getCostCapPolicyErr
	}
	return connect.NewResponse(f.getCostCapPolicyResp), nil
}

func (f *fakeIntegrationsClient) SetCostCapPolicy(_ context.Context, req *connect.Request[pidgrv1.SetCostCapPolicyRequest]) (*connect.Response[pidgrv1.SetCostCapPolicyResponse], error) {
	f.setCostCapPolicyReq = req.Msg
	if f.setCostCapPolicyErr != nil {
		return nil, f.setCostCapPolicyErr
	}
	return connect.NewResponse(f.setCostCapPolicyResp), nil
}

// fakeCampaignClient is a minimal stub of pidgrv1connect.CampaignServiceClient used to
// exercise the GetCampaignArchetypeBreakdown handler in isolation.
type fakeCampaignClient struct {
	pidgrv1connect.CampaignServiceClient

	getCampaignArchetypeBreakdownResp *pidgrv1.GetCampaignArchetypeBreakdownResponse
	getCampaignArchetypeBreakdownErr  error
	getCampaignArchetypeBreakdownReq  *pidgrv1.GetCampaignArchetypeBreakdownRequest
}

func (f *fakeCampaignClient) GetCampaignArchetypeBreakdown(_ context.Context, req *connect.Request[pidgrv1.GetCampaignArchetypeBreakdownRequest]) (*connect.Response[pidgrv1.GetCampaignArchetypeBreakdownResponse], error) {
	f.getCampaignArchetypeBreakdownReq = req.Msg
	if f.getCampaignArchetypeBreakdownErr != nil {
		return nil, f.getCampaignArchetypeBreakdownErr
	}
	return connect.NewResponse(f.getCampaignArchetypeBreakdownResp), nil
}

// callTool spins up an in-memory MCP server with only the integrations + campaign tool
// registrations against a hand-built *transport.Clients (with the supplied fake
// IntegrationsService client and an optional fake CampaignService client).
func callTool(t *testing.T, fake *fakeIntegrationsClient, campaign *fakeCampaignClient, toolName string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	clients := &transport.Clients{Integrations: fake}
	if campaign != nil {
		clients.Campaigns = campaign
	} else {
		// Some tools (the integrations ones) don't use Campaigns; supply a stub.
		clients.Campaigns = &fakeCampaignClient{}
	}
	registerIntegrationsTools(server, clients)
	registerCampaignArchetypeBreakdownTool(server, clients)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	st, ct := mcp.NewInMemoryTransports()
	go func() { _ = server.Run(context.Background(), st) }()

	session, err := client.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", toolName, err)
	}
	return result
}

func resultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// ─── list_reachabilities_for_user ───────────────────────────────────────────

func TestListReachabilitiesForUserHappyPath(t *testing.T) {
	fake := &fakeIntegrationsClient{
		listReachabilityForUserResp: &pidgrv1.ListReachabilityForUserResponse{
			Reachabilities: []*pidgrv1.Reachability{
				{
					Id:      "reach-1",
					OrgId:   "org-1",
					UserId:  "user-1",
					Channel: pidgrv1.ChannelName_CHANNEL_NAME_EMAIL,
				},
			},
		},
	}
	result := callTool(t, fake, nil, "list_reachabilities_for_user", map[string]any{
		"org_id":  "org-1",
		"user_id": "user-1",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(result))
	}
	if fake.listReachabilityForUserReq.GetOrgId() != "org-1" {
		t.Errorf("expected org_id forwarded, got %q", fake.listReachabilityForUserReq.GetOrgId())
	}
	if fake.listReachabilityForUserReq.GetUserId() != "user-1" {
		t.Errorf("expected user_id forwarded, got %q", fake.listReachabilityForUserReq.GetUserId())
	}
	if !strings.Contains(resultText(result), "reach-1") {
		t.Errorf("expected response JSON to include reach-1, got %s", resultText(result))
	}
}

func TestListReachabilitiesForUserPermissionDenied(t *testing.T) {
	fake := &fakeIntegrationsClient{
		listReachabilityForUserErr: connect.NewError(connect.CodePermissionDenied, errors.New("cross-org access")),
	}
	result := callTool(t, fake, nil, "list_reachabilities_for_user", map[string]any{
		"org_id":  "other-org",
		"user_id": "user-1",
	})
	if !result.IsError {
		t.Fatal("expected IsError=true for permission_denied")
	}
	if !strings.Contains(strings.ToLower(resultText(result)), "permission denied") {
		t.Errorf("expected permission-denied message, got %q", resultText(result))
	}
}

func TestListReachabilitiesForUserInternalError(t *testing.T) {
	fake := &fakeIntegrationsClient{
		listReachabilityForUserErr: connect.NewError(connect.CodeInternal, errors.New("db down")),
	}
	result := callTool(t, fake, nil, "list_reachabilities_for_user", map[string]any{
		"org_id":  "org-1",
		"user_id": "user-1",
	})
	if !result.IsError {
		t.Fatal("expected IsError=true for internal")
	}
	if !strings.Contains(strings.ToLower(resultText(result)), "internal") {
		t.Errorf("expected internal error message, got %q", resultText(result))
	}
}

// ─── upsert_reachability ────────────────────────────────────────────────────

func TestUpsertReachabilityHappyPath(t *testing.T) {
	fake := &fakeIntegrationsClient{
		upsertReachabilityResp: &pidgrv1.UpsertReachabilityResponse{
			Reachability: &pidgrv1.Reachability{
				Id:      "reach-1",
				OrgId:   "org-1",
				UserId:  "user-1",
				Channel: pidgrv1.ChannelName_CHANNEL_NAME_EMAIL,
			},
		},
	}
	result := callTool(t, fake, nil, "upsert_reachability", map[string]any{
		"org_id":               "org-1",
		"user_id":              "user-1",
		"channel":              "CHANNEL_NAME_EMAIL",
		"identifier_plaintext": "alice@example.com",
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", resultText(result))
	}
	if fake.upsertReachabilityReq.GetChannel() != pidgrv1.ChannelName_CHANNEL_NAME_EMAIL {
		t.Errorf("expected channel forwarded as EMAIL, got %v", fake.upsertReachabilityReq.GetChannel())
	}
	if fake.upsertReachabilityReq.GetIdentifierPlaintext() != "alice@example.com" {
		t.Errorf("expected identifier_plaintext forwarded verbatim, got %q", fake.upsertReachabilityReq.GetIdentifierPlaintext())
	}
}

func TestUpsertReachabilityWithRegionConstraint(t *testing.T) {
	fake := &fakeIntegrationsClient{
		upsertReachabilityResp: &pidgrv1.UpsertReachabilityResponse{
			Reachability: &pidgrv1.Reachability{Id: "reach-1"},
		},
	}
	result := callTool(t, fake, nil, "upsert_reachability", map[string]any{
		"org_id":               "org-1",
		"user_id":              "user-1",
		"channel":              "CHANNEL_NAME_SMS",
		"identifier_plaintext": "+11235550100",
		"region_constraint":    "us-east-1",
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", resultText(result))
	}
	if fake.upsertReachabilityReq.RegionConstraint == nil || *fake.upsertReachabilityReq.RegionConstraint != "us-east-1" {
		t.Errorf("expected region_constraint forwarded, got %v", fake.upsertReachabilityReq.RegionConstraint)
	}
}

func TestUpsertReachabilityUnknownChannel(t *testing.T) {
	fake := &fakeIntegrationsClient{}
	result := callTool(t, fake, nil, "upsert_reachability", map[string]any{
		"org_id":               "org-1",
		"user_id":              "user-1",
		"channel":              "CHANNEL_NAME_FAX",
		"identifier_plaintext": "anything",
	})
	if !result.IsError {
		t.Fatal("expected IsError=true for unknown channel")
	}
	if !strings.Contains(strings.ToLower(resultText(result)), "invalid") {
		t.Errorf("expected invalid-argument message, got %q", resultText(result))
	}
	// The upstream MUST NOT be called for invalid channel names.
	if fake.upsertReachabilityReq != nil {
		t.Error("expected upstream NOT to be called when channel is invalid")
	}
}

// TestUpsertReachabilityDoesNotLogPlaintext is the task 4.4 regression guard:
// pidgr-mcp MUST NOT emit the sensitive identifier_plaintext value into any log
// sink. We swap the process-default slog logger for a buffer-backed handler at
// debug level (the most verbose level), drive upsert_reachability with a sentinel
// identifier, and assert the sentinel never lands in the captured log output.
func TestUpsertReachabilityDoesNotLogPlaintext(t *testing.T) {
	const sentinel = "SENSITIVE-SENTINEL-alice@example.com"

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	fake := &fakeIntegrationsClient{
		upsertReachabilityResp: &pidgrv1.UpsertReachabilityResponse{
			Reachability: &pidgrv1.Reachability{Id: "reach-1", Channel: pidgrv1.ChannelName_CHANNEL_NAME_EMAIL},
		},
	}
	result := callTool(t, fake, nil, "upsert_reachability", map[string]any{
		"org_id":               "org-1",
		"user_id":              "user-1",
		"channel":              "CHANNEL_NAME_EMAIL",
		"identifier_plaintext": sentinel,
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", resultText(result))
	}
	// Sanity: the plaintext IS forwarded to the upstream RPC (it just must not be logged).
	if fake.upsertReachabilityReq.GetIdentifierPlaintext() != sentinel {
		t.Fatalf("expected identifier_plaintext forwarded verbatim, got %q", fake.upsertReachabilityReq.GetIdentifierPlaintext())
	}
	if bytes.Contains(buf.Bytes(), []byte(sentinel)) {
		t.Errorf("identifier_plaintext sentinel leaked into log output:\n%s", buf.String())
	}
}

// ─── remove_reachability ────────────────────────────────────────────────────

func TestRemoveReachabilityHappyPath(t *testing.T) {
	fake := &fakeIntegrationsClient{
		removeReachabilityResp: &pidgrv1.RemoveReachabilityResponse{Removed: true},
	}
	result := callTool(t, fake, nil, "remove_reachability", map[string]any{
		"org_id":  "org-1",
		"user_id": "user-1",
		"channel": "CHANNEL_NAME_EMAIL",
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", resultText(result))
	}
	if !strings.Contains(resultText(result), "removed") {
		t.Errorf("expected response to mention removed, got %s", resultText(result))
	}
}

func TestRemoveReachabilityIdempotent(t *testing.T) {
	fake := &fakeIntegrationsClient{
		removeReachabilityResp: &pidgrv1.RemoveReachabilityResponse{Removed: false},
	}
	result := callTool(t, fake, nil, "remove_reachability", map[string]any{
		"org_id":  "org-1",
		"user_id": "user-1",
		"channel": "CHANNEL_NAME_EMAIL",
	})
	if result.IsError {
		t.Fatalf("expected success (idempotent), got %s", resultText(result))
	}
}

func TestRemoveReachabilityNotFound(t *testing.T) {
	// Real upstream never returns NOT_FOUND (idempotent), but we exercise the
	// mapping so future regressions are surfaced if the contract slips.
	fake := &fakeIntegrationsClient{
		removeReachabilityErr: connect.NewError(connect.CodeNotFound, errors.New("missing")),
	}
	result := callTool(t, fake, nil, "remove_reachability", map[string]any{
		"org_id":  "org-1",
		"user_id": "user-1",
		"channel": "CHANNEL_NAME_EMAIL",
	})
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(strings.ToLower(resultText(result)), "not found") {
		t.Errorf("expected not-found message, got %q", resultText(result))
	}
}

// ─── get_cost_cap_policy ────────────────────────────────────────────────────

func TestGetCostCapPolicyHappyPath(t *testing.T) {
	fake := &fakeIntegrationsClient{
		getCostCapPolicyResp: &pidgrv1.GetCostCapPolicyResponse{
			OrgId:        "org-1",
			Channel:      pidgrv1.ChannelName_CHANNEL_NAME_EMAIL,
			CapMicros:    50_000_000,
			UsedMicros:   12_345_678,
			PeriodYyyymm: 202605,
		},
	}
	result := callTool(t, fake, nil, "get_cost_cap_policy", map[string]any{
		"org_id":  "org-1",
		"channel": "CHANNEL_NAME_EMAIL",
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", resultText(result))
	}
	body := resultText(result)
	if !strings.Contains(body, "capMicros") && !strings.Contains(body, "cap_micros") {
		t.Errorf("expected cap_micros field in response, got %s", body)
	}
}

func TestGetCostCapPolicyUnknownChannel(t *testing.T) {
	fake := &fakeIntegrationsClient{}
	result := callTool(t, fake, nil, "get_cost_cap_policy", map[string]any{
		"org_id":  "org-1",
		"channel": "CHANNEL_NAME_BOGUS",
	})
	if !result.IsError {
		t.Fatal("expected IsError=true for unknown channel")
	}
	if fake.getCostCapPolicyReq != nil {
		t.Error("expected upstream NOT to be called when channel is invalid")
	}
}

// ─── set_cost_cap_policy ────────────────────────────────────────────────────

func TestSetCostCapPolicyHappyPath(t *testing.T) {
	fake := &fakeIntegrationsClient{
		setCostCapPolicyResp: &pidgrv1.SetCostCapPolicyResponse{
			OrgId:        "org-1",
			Channel:      pidgrv1.ChannelName_CHANNEL_NAME_EMAIL,
			CapMicros:    100_000_000,
			UsedMicros:   0,
			PeriodYyyymm: 202605,
		},
	}
	result := callTool(t, fake, nil, "set_cost_cap_policy", map[string]any{
		"org_id":     "org-1",
		"channel":    "CHANNEL_NAME_EMAIL",
		"cap_micros": int64(100_000_000),
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", resultText(result))
	}
	if fake.setCostCapPolicyReq.GetCapMicros() != 100_000_000 {
		t.Errorf("expected cap_micros 100000000, got %d", fake.setCostCapPolicyReq.GetCapMicros())
	}
}

func TestSetCostCapPolicyPermissionDenied(t *testing.T) {
	fake := &fakeIntegrationsClient{
		setCostCapPolicyErr: connect.NewError(connect.CodePermissionDenied, errors.New("admin only")),
	}
	result := callTool(t, fake, nil, "set_cost_cap_policy", map[string]any{
		"org_id":     "org-1",
		"channel":    "CHANNEL_NAME_EMAIL",
		"cap_micros": int64(100_000_000),
	})
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(strings.ToLower(resultText(result)), "permission denied") {
		t.Errorf("expected permission-denied message, got %q", resultText(result))
	}
}

// ─── get_campaign_archetype_breakdown ───────────────────────────────────────

func TestGetCampaignArchetypeBreakdownHappyPath(t *testing.T) {
	campaign := &fakeCampaignClient{
		getCampaignArchetypeBreakdownResp: &pidgrv1.GetCampaignArchetypeBreakdownResponse{
			Shifts: []*pidgrv1.ArchetypeShareShift{
				{
					EmailDeliveredCount: 100,
					EmailOpenRateReal:   0.42,
					EmailOpenRateRaw:    0.5,
				},
			},
			InsufficientHistory: false,
		},
	}
	result := callTool(t, &fakeIntegrationsClient{}, campaign, "get_campaign_archetype_breakdown", map[string]any{
		"campaign_id": "camp-1",
	})
	if result.IsError {
		t.Fatalf("expected success, got %s", resultText(result))
	}
	if campaign.getCampaignArchetypeBreakdownReq.GetCampaignId() != "camp-1" {
		t.Errorf("expected campaign_id forwarded, got %q", campaign.getCampaignArchetypeBreakdownReq.GetCampaignId())
	}
	body := resultText(result)
	// Email-engagement fields must surface verbatim in proto-JSON.
	if !strings.Contains(body, "emailDeliveredCount") && !strings.Contains(body, "email_delivered_count") {
		t.Errorf("expected email_delivered_count in response, got %s", body)
	}
	if !strings.Contains(body, "emailOpenRateReal") && !strings.Contains(body, "email_open_rate_real") {
		t.Errorf("expected email_open_rate_real in response, got %s", body)
	}
}

func TestGetCampaignArchetypeBreakdownNotFound(t *testing.T) {
	campaign := &fakeCampaignClient{
		getCampaignArchetypeBreakdownErr: connect.NewError(connect.CodeNotFound, errors.New("no such campaign")),
	}
	result := callTool(t, &fakeIntegrationsClient{}, campaign, "get_campaign_archetype_breakdown", map[string]any{
		"campaign_id": "missing",
	})
	if !result.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(strings.ToLower(resultText(result)), "not found") {
		t.Errorf("expected not-found message, got %q", resultText(result))
	}
}

func TestGetCampaignArchetypeBreakdownInsufficientHistory(t *testing.T) {
	campaign := &fakeCampaignClient{
		getCampaignArchetypeBreakdownResp: &pidgrv1.GetCampaignArchetypeBreakdownResponse{
			Shifts:              nil,
			InsufficientHistory: true,
		},
	}
	result := callTool(t, &fakeIntegrationsClient{}, campaign, "get_campaign_archetype_breakdown", map[string]any{
		"campaign_id": "camp-1",
	})
	if result.IsError {
		t.Fatalf("expected success (insufficient_history is not an error), got %s", resultText(result))
	}
	body := resultText(result)
	if !strings.Contains(body, "insufficientHistory") && !strings.Contains(body, "insufficient_history") {
		t.Errorf("expected insufficient_history surfaced as success payload, got %s", body)
	}
}

// ─── Integrations client wiring on transport.Clients ────────────────────────

// This test simply asserts that the integrations field is populated by both
// transport constructors — full wire behavior is covered by transport/.
func TestIntegrationsClientPopulatedByTransport(t *testing.T) {
	oauthClients := transport.NewOAuthClients("http://localhost:50051", func(context.Context) (string, error) {
		return "test-token", nil
	}, nil)
	if oauthClients.Integrations == nil {
		t.Error("NewOAuthClients did not set Integrations client")
	}
	dynamicClients := transport.NewDynamicTokenClients("http://localhost:50051")
	if dynamicClients.Integrations == nil {
		t.Error("NewDynamicTokenClients did not set Integrations client")
	}
}

// ─── Description-only smoke check (parse the JSON inputSchema) ──────────────

// TestIntegrationsToolsHaveDescriptions guards against accidental description regressions.
func TestIntegrationsToolsHaveDescriptions(t *testing.T) {
	tools := listTools(t)
	names := []string{
		"list_reachabilities_for_user",
		"upsert_reachability",
		"remove_reachability",
		"get_cost_cap_policy",
		"set_cost_cap_policy",
		"get_campaign_archetype_breakdown",
	}
	for _, name := range names {
		tool := toolByName(tools, name)
		if tool == nil {
			t.Errorf("%s not registered", name)
			continue
		}
		if tool.Description == "" {
			t.Errorf("%s missing description", name)
		}
		// Sanity: schema is valid JSON.
		b, _ := json.Marshal(tool.InputSchema)
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Errorf("%s: schema not valid JSON: %v", name, err)
		}
	}
}
