// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pidgr/pidgr-mcp/internal/transport"
)

func listTools(t *testing.T) []*mcp.Tool {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	clients := transport.NewStaticTokenClients("http://localhost:50051", "test-key")
	RegisterAll(server, clients)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	st, ct := mcp.NewInMemoryTransports()
	go func() { _ = server.Run(context.Background(), st) }()

	session, err := client.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	return result.Tools
}

func toolByName(tools []*mcp.Tool, name string) *mcp.Tool {
	for _, t := range tools {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func schemaHasProperty(t *testing.T, tool *mcp.Tool, prop string) bool {
	t.Helper()
	b, _ := json.Marshal(tool.InputSchema)
	var schema map[string]any
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Errorf("tool %q: failed to parse schema: %v", tool.Name, err)
		return false
	}
	props, _ := schema["properties"].(map[string]any)
	_, ok := props[prop]
	return ok
}

func TestAllToolsHaveDescriptions(t *testing.T) {
	for _, tool := range listTools(t) {
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
	}
}

func TestPrivacyToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{
		"export_user_data", "delete_user_data", "cancel_deletion",
		"immediate_delete", "list_privacy_requests",
		"rectify_user_data", "restrict_processing",
	}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("privacy tool %q not registered", name)
		}
	}
}

func TestAuditToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{"list_audit_events", "export_audit_trail", "list_audit_exports"}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("audit tool %q not registered", name)
		}
	}
}

func TestSSOToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{"create_sso_provider", "get_sso_provider", "delete_sso_provider"}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("SSO tool %q not registered", name)
		}
	}
}

func TestInviteLinkToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{"create_invite_link", "list_invite_links", "revoke_invite_link"}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("invite link tool %q not registered", name)
		}
	}
}

func TestDeviceToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{"list_devices", "list_member_devices"}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("device tool %q not registered", name)
		}
	}
}

func TestNewMemberToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{"bulk_invite_users", "get_user_settings", "update_user_settings"}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("member tool %q not registered", name)
		}
	}
}

func TestPrivacyToolsRequireUserID(t *testing.T) {
	tools := listTools(t)
	for _, name := range []string{"export_user_data", "delete_user_data", "rectify_user_data", "restrict_processing"} {
		tool := toolByName(tools, name)
		if tool == nil {
			t.Errorf("tool %q not found", name)
			continue
		}
		if !schemaHasProperty(t, tool, "user_id") {
			t.Errorf("tool %q missing user_id property", name)
		}
	}
}

func TestDeleteUserDataHasAnonymize(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "delete_user_data")
	if tool == nil {
		t.Fatal("delete_user_data not found")
		return
	}
	if !schemaHasProperty(t, tool, "anonymize") {
		t.Error("delete_user_data missing anonymize property")
	}
}

func TestExportAuditTrailHasFormat(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "export_audit_trail")
	if tool == nil {
		t.Fatal("export_audit_trail not found")
		return
	}
	if !schemaHasProperty(t, tool, "format") {
		t.Error("export_audit_trail missing format property")
	}
}

func TestCancelDeletionRequiresConfirmation(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "cancel_deletion")
	if tool == nil {
		t.Fatal("cancel_deletion not found")
		return
	}
	if !schemaHasProperty(t, tool, "request_id") {
		t.Error("cancel_deletion missing request_id")
	}
	if !schemaHasProperty(t, tool, "confirm_email") {
		t.Error("cancel_deletion missing confirm_email")
	}
}

func TestBulkInviteHasEmails(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "bulk_invite_users")
	if tool == nil {
		t.Fatal("bulk_invite_users not found")
		return
	}
	if !schemaHasProperty(t, tool, "emails") {
		t.Error("bulk_invite_users missing emails property")
	}
}

func TestCreateInviteLinkHasExpiresInHours(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "create_invite_link")
	if tool == nil {
		t.Fatal("create_invite_link not found")
		return
	}
	if !schemaHasProperty(t, tool, "expires_in_hours") {
		t.Error("create_invite_link missing expires_in_hours property")
	}
}

func TestListMemberDevicesRequiresUserID(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "list_member_devices")
	if tool == nil {
		t.Fatal("list_member_devices not found")
		return
	}
	if !schemaHasProperty(t, tool, "user_id") {
		t.Error("list_member_devices missing user_id property")
	}
}

func TestInviteUserHasDataGovernanceRegion(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "invite_user")
	if tool == nil {
		t.Fatal("invite_user not found")
		return
	}
	if !schemaHasProperty(t, tool, "data_governance_region") {
		t.Error("invite_user missing data_governance_region property")
	}
}

func TestCreateInviteLinkHasDataGovernanceRegion(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "create_invite_link")
	if tool == nil {
		t.Fatal("create_invite_link not found")
		return
	}
	if !schemaHasProperty(t, tool, "data_governance_region") {
		t.Error("create_invite_link missing data_governance_region property")
	}
}

func TestUpdateUserRegionRegistered(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "update_user_region")
	if tool == nil {
		t.Fatal("update_user_region not registered")
		return
	}
	if !schemaHasProperty(t, tool, "user_id") {
		t.Error("update_user_region missing user_id property")
	}
	if !schemaHasProperty(t, tool, "data_governance_region") {
		t.Error("update_user_region missing data_governance_region property")
	}
}

func TestListUserOrganizationsRegistered(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "list_user_organizations")
	if tool == nil {
		t.Fatal("list_user_organizations not registered")
		return
	}
	if tool.Description == "" {
		t.Error("list_user_organizations has empty description")
	}
}

func TestListMyPrivacyRequestsRegistered(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "list_my_privacy_requests")
	if tool == nil {
		t.Fatal("list_my_privacy_requests not registered")
		return
	}
	if !schemaHasProperty(t, tool, "page_size") {
		t.Error("list_my_privacy_requests missing page_size property")
	}
	if !schemaHasProperty(t, tool, "request_type") {
		t.Error("list_my_privacy_requests missing request_type property")
	}
	if !schemaHasProperty(t, tool, "status") {
		t.Error("list_my_privacy_requests missing status property")
	}
}

func TestUpdateTemplateTranslationRegistered(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "update_template_translation")
	if tool == nil {
		t.Fatal("update_template_translation not registered")
		return
	}
	if !schemaHasProperty(t, tool, "translation_id") {
		t.Error("update_template_translation missing translation_id property")
	}
	if !schemaHasProperty(t, tool, "title") {
		t.Error("update_template_translation missing title property")
	}
	if !schemaHasProperty(t, tool, "body") {
		t.Error("update_template_translation missing body property")
	}
	if !schemaHasProperty(t, tool, "status") {
		t.Error("update_template_translation missing status property")
	}
}
