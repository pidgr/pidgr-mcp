// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pidgr/pidgr-mcp/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAPIServer records all incoming RPC calls and returns empty successful responses.
type mockAPIServer struct {
	mu    sync.Mutex
	calls []apiCall
}

type apiCall struct {
	Method string
	Path   string
	Body   string
}

func (m *mockAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, 0)
	if r.Body != nil {
		body, _ = readBody(r)
	}

	m.mu.Lock()
	m.calls = append(m.calls, apiCall{
		Method: r.Method,
		Path:   r.URL.Path,
		Body:   string(body),
	})
	m.mu.Unlock()

	// Return a valid Connect-RPC response with empty JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{}"))
}

func (m *mockAPIServer) getCalls() []apiCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]apiCall, len(m.calls))
	copy(result, m.calls)
	return result
}

func (m *mockAPIServer) lastCallPath() string {
	calls := m.getCalls()
	if len(calls) == 0 {
		return ""
	}
	return calls[len(calls)-1].Path
}

func (m *mockAPIServer) lastCallBody() string {
	calls := m.getCalls()
	if len(calls) == 0 {
		return ""
	}
	return calls[len(calls)-1].Body
}

func (m *mockAPIServer) reset() {
	m.mu.Lock()
	m.calls = nil
	m.mu.Unlock()
}

func readBody(r *http.Request) ([]byte, error) {
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 256)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

// setupToolTest creates a mock API server, registers all MCP tools, and returns
// a function to call tools by name with JSON input.
func setupToolTest(t *testing.T) (callTool func(name string, args map[string]interface{}) (*mcp.CallToolResult, error), mock *mockAPIServer) {
	t.Helper()

	mock = &mockAPIServer{}
	ts := httptest.NewServer(mock)
	t.Cleanup(ts.Close)

	clients := transport.NewStaticTokenClients(ts.URL, "test-key")

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pidgr-test",
		Version: "test",
	}, nil)
	RegisterAll(server, clients)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "test",
	}, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go func() { _ = server.Run(context.Background(), serverTransport) }()

	session, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	callTool = func(name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
		mock.reset()
		argsJSON, _ := json.Marshal(args)
		return session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      name,
			Arguments: unmarshalArgs(argsJSON),
		})
	}

	return callTool, mock
}

func unmarshalArgs(data []byte) map[string]any {
	var args map[string]any
	_ = json.Unmarshal(data, &args)
	return args
}

// ─── Tool execution tests ───────────────────────────────────────────────────

func TestToolExecution_ListRoles(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_roles", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.RoleService/ListRoles"))
}

func TestToolExecution_CreateRole(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("create_role", map[string]interface{}{
		"name":        "Team Lead",
		"permissions": []string{"PERMISSION_CAMPAIGNS_READ"},
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.RoleService/CreateRole"))
	assert.Contains(t, mock.lastCallBody(), "Team Lead")
}

func TestToolExecution_GetOrganization(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("get_organization", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.OrganizationService/GetOrganization"))
}

func TestToolExecution_ListCampaigns(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_campaigns", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.CampaignService/ListCampaigns"))
}

func TestToolExecution_CreateCampaign(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("create_campaign", map[string]interface{}{
		"name":             "Q1 Update",
		"template_id":      "tpl-1",
		"template_version": 1,
		"sender_name":      "HR",
		"title":            "Benefits",
		"user_ids":         []string{"u1"},
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.CampaignService/CreateCampaign"))
}

func TestToolExecution_ListUsers(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_users", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.MemberService/ListUsers"))
}

func TestToolExecution_InviteUser(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("invite_user", map[string]interface{}{
		"email":   "new@example.com",
		"role_id": "role-1",
		"name":    "New User",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.MemberService/InviteUser"))
}

func TestToolExecution_ListTemplates(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_templates", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.TemplateService/ListTemplates"))
}

func TestToolExecution_ListGroups(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_groups", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.GroupService/ListGroups"))
}

func TestToolExecution_ListTeams(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_teams", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.TeamService/ListTeams"))
}

func TestToolExecution_ListApiKeys(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_api_keys", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.ApiKeyService/ListApiKeys"))
}

func TestToolExecution_QueryHeatmapData(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("query_heatmap_data", map[string]interface{}{
		"screen_name": "HomeScreen",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.HeatmapService/QueryHeatmapData"))
}

func TestToolExecution_ListSessionRecordings(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_session_recordings", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.ReplayService/ListSessionRecordings"))
}

func TestToolExecution_ExportUserData(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("export_user_data", map[string]interface{}{
		"user_id": "user@example.com",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.PrivacyService/ExportUserData"))
}

func TestToolExecution_ListAuditEvents(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_audit_events", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.AuditService/ListAuditEvents"))
}

func TestToolExecution_ListDevices(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("list_devices", nil)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.DeviceService/ListDevices"))
}

func TestToolExecution_CreateInviteLink(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("create_invite_link", map[string]interface{}{
		"role_id":          "role-1",
		"max_uses":         10,
		"expires_in_hours": 24,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.InviteLinkService/CreateInviteLink"))
}

func TestToolExecution_GetDataExistenceConfirmation(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("get_data_existence_confirmation", map[string]interface{}{
		"user_id": "user@example.com",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.PrivacyService/GetDataExistenceConfirmation"))
}

func TestToolExecution_RotateAnalyticsSalt(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("rotate_analytics_salt", map[string]interface{}{
		"new_bucket_count": 25,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.OrganizationService/RotateAnalyticsSalt"))
}

func TestToolExecution_UpdateAnalyticsEpsilon(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("update_analytics_epsilon", map[string]interface{}{
		"epsilon": 1.5,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.OrganizationService/UpdateAnalyticsEpsilon"))
}

func TestToolExecution_RestrictProcessing(t *testing.T) {
	callTool, mock := setupToolTest(t)
	_, err := callTool("restrict_processing", map[string]interface{}{
		"user_id": "user@example.com",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(mock.lastCallPath(), "/pidgr.v1.PrivacyService/RestrictProcessing"))
}

// ─── Error handling tests ───────────────────────────────────────────────────

func TestToolExecution_APIError_ReturnsErrorResult(t *testing.T) {
	// Create a server that returns Connect errors
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"not_found","message":"resource not found"}`))
	}))
	t.Cleanup(errServer.Close)

	clients := transport.NewStaticTokenClients(errServer.URL, "test-key")

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	RegisterAll(server, clients)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	st, ct := mcp.NewInMemoryTransports()
	go func() { _ = server.Run(context.Background(), st) }()

	session, err := client.Connect(context.Background(), ct, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }()

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "list_roles",
	})

	require.NoError(t, err, "MCP call should succeed even when API returns error")
	assert.True(t, result.IsError, "result should be marked as error")
}

// ─── All 75 tools callable ──────────────────────────────────────────────────

func TestAllToolsAreCallable(t *testing.T) {
	mock := &mockAPIServer{}
	ts := httptest.NewServer(mock)
	t.Cleanup(ts.Close)

	clients := transport.NewStaticTokenClients(ts.URL, "test-key")

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	RegisterAll(server, clients)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	st, ct := mcp.NewInMemoryTransports()
	go func() { _ = server.Run(context.Background(), st) }()

	session, err := client.Connect(context.Background(), ct, nil)
	require.NoError(t, err)
	defer func() { _ = session.Close() }()

	toolList, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	for _, tool := range toolList.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			mock.reset()
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name: tool.Name,
			})
			// Tools with required fields will fail schema validation — that's
			// expected. What we're checking is that no tool panics or crashes.
			if err != nil {
				assert.Contains(t, err.Error(), "invalid params", "tool %s error should be schema validation, not a crash", tool.Name)
				return
			}
			assert.NotNil(t, result, "tool %s should return a result", tool.Name)
		})
	}
}
