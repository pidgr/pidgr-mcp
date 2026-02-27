// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pidgr/pidgr-mcp/internal/transport"
)

func TestRegisterAll(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pidgr-test",
		Version: "test",
	}, nil)

	// Create clients with a dummy URL (we won't actually make calls).
	clients := transport.NewStaticTokenClients("http://localhost:50051", "test-key")

	// Register all tools.
	RegisterAll(server, clients)

	// Use a test client to list tools.
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "test",
	}, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go server.Run(context.Background(), serverTransport)

	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	want := 49
	if got := len(result.Tools); got != want {
		t.Errorf("RegisterAll registered %d tools, want %d", got, want)
		for _, tool := range result.Tools {
			t.Logf("  - %s", tool.Name)
		}
	}
}

func TestToolNames(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pidgr-test",
		Version: "test",
	}, nil)

	clients := transport.NewStaticTokenClients("http://localhost:50051", "test-key")
	RegisterAll(server, clients)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "test",
	}, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go server.Run(context.Background(), serverTransport)

	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	expectedTools := []string{
		// Campaign (7)
		"create_campaign", "update_campaign", "start_campaign", "get_campaign",
		"list_campaigns", "cancel_campaign", "list_deliveries",
		// Template (4)
		"create_template", "update_template", "get_template", "list_templates",
		// Group (9)
		"create_group", "get_group", "list_groups", "update_group", "delete_group",
		"add_group_members", "remove_group_members", "list_group_members", "get_user_group_memberships",
		// Team (8)
		"create_team", "get_team", "list_teams", "update_team", "delete_team",
		"add_team_members", "remove_team_members", "list_team_members",
		// Member (6)
		"invite_user", "get_user", "list_users", "update_user_role", "deactivate_user", "update_user_profile",
		// Organization (4)
		"create_organization", "get_organization", "update_organization", "update_sso_attribute_mappings",
		// Role (4)
		"list_roles", "create_role", "update_role", "delete_role",
		// ApiKey (3)
		"create_api_key", "list_api_keys", "revoke_api_key",
		// Heatmap (2)
		"query_heatmap_data", "list_screenshots",
		// Replay (2)
		"list_session_recordings", "get_session_snapshots",
	}

	registered := make(map[string]bool)
	for _, tool := range result.Tools {
		registered[tool.Name] = true
	}

	for _, name := range expectedTools {
		if !registered[name] {
			t.Errorf("expected tool %q not registered", name)
		}
	}
}
