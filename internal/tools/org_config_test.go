// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"testing"
)

func TestOrgPrivacySettingsToolsSchema(t *testing.T) {
	tools := listTools(t)

	get := toolByName(tools, "get_org_privacy_settings")
	if get == nil {
		t.Fatal("get_org_privacy_settings not registered")
	}
	if get.Description == "" {
		t.Error("get_org_privacy_settings has empty description")
	}

	update := toolByName(tools, "update_org_privacy_settings")
	if update == nil {
		t.Fatal("update_org_privacy_settings not registered")
	}
	for _, prop := range []string{
		"ai_clustering_enabled",
		"behavioral_analytics_enabled",
		"third_party_channels_enabled",
	} {
		if !schemaHasProperty(t, update, prop) {
			t.Errorf("update_org_privacy_settings missing %s property", prop)
		}
	}
	// Every toggle is tri-state: omitting it leaves the setting unchanged, so
	// none may be marked required or a caller could not flip one in isolation.
	if names := schemaRequired(t, update); len(names) != 0 {
		t.Errorf("update_org_privacy_settings must mark no property required, got %v", names)
	}

	// behavioral_analytics_enabled is the E11 gate that silently drops tap
	// ingestion and suppresses the Compass heatmap, so the description has to
	// say so — an agent debugging an empty heatmap reads this first.
	if !containsFold(update.Description, "behavioral") {
		t.Errorf("update_org_privacy_settings description should name the behavioral-analytics gate, got: %s", update.Description)
	}
}

func TestDeleteSandboxOrganizationSchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "delete_sandbox_organization")
	if tool == nil {
		t.Fatal("delete_sandbox_organization not registered")
	}
	if !schemaHasProperty(t, tool, "org_id") {
		t.Error("delete_sandbox_organization missing org_id property")
	}
	if names := schemaRequired(t, tool); !contains(names, "org_id") {
		t.Errorf("delete_sandbox_organization must mark org_id required, got %v", names)
	}
	// Deleting the wrong tenant is unrecoverable; the description must say the
	// scope is sandbox-only and the delete is permanent.
	if !containsFold(tool.Description, "sandbox") || !containsFold(tool.Description, "permanent") {
		t.Errorf("delete_sandbox_organization description must state sandbox-only and permanent, got: %s", tool.Description)
	}
}

func TestTriggerArchetypeClusteringSchema(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "trigger_archetype_clustering")
	if tool == nil {
		t.Fatal("trigger_archetype_clustering not registered")
	}
	if !schemaHasProperty(t, tool, "group_id") {
		t.Error("trigger_archetype_clustering missing group_id property")
	}
	if names := schemaRequired(t, tool); !contains(names, "group_id") {
		t.Errorf("trigger_archetype_clustering must mark group_id required, got %v", names)
	}
	// It shares the monthly manual-retrain quota with trigger_ml_pipeline, and
	// picking the wrong one burns a scarce allowance — the description has to
	// draw the distinction.
	if !containsFold(tool.Description, "quota") {
		t.Errorf("trigger_archetype_clustering description should mention the shared monthly quota, got: %s", tool.Description)
	}
}

func TestOrgScopedPrivacyReadToolsSchema(t *testing.T) {
	tools := listTools(t)

	export := toolByName(tools, "export_org_data")
	if export == nil {
		t.Fatal("export_org_data not registered")
	}
	if export.Description == "" {
		t.Error("export_org_data has empty description")
	}

	incidents := toolByName(tools, "list_org_security_incidents")
	if incidents == nil {
		t.Fatal("list_org_security_incidents not registered")
	}
	for _, prop := range []string{"page_size", "page_token"} {
		if !schemaHasProperty(t, incidents, prop) {
			t.Errorf("list_org_security_incidents missing %s property", prop)
		}
	}
}

func TestOrgIntegrationsConfigToolsSchema(t *testing.T) {
	tools := listTools(t)

	for _, tc := range []struct {
		name  string
		props []string
	}{
		{"get_org_webhook_config", []string{"org_id"}},
		{"set_org_webhook_config", []string{"org_id", "url", "enabled", "secret"}},
		{"get_region_policy", []string{"org_id", "channel"}},
		{"set_region_policy", []string{"org_id", "channel", "allowed_regions"}},
	} {
		tool := toolByName(tools, tc.name)
		if tool == nil {
			t.Errorf("%s not registered", tc.name)
			continue
		}
		if tool.Description == "" {
			t.Errorf("%s has empty description", tc.name)
		}
		for _, prop := range tc.props {
			if !schemaHasProperty(t, tool, prop) {
				t.Errorf("%s missing %s property", tc.name, prop)
			}
		}
	}

	// The webhook secret is write-only and rotating it invalidates the
	// receiver's verification, so omitting it must keep the current secret —
	// it can never be required.
	set := toolByName(tools, "set_org_webhook_config")
	if set == nil {
		t.Fatal("set_org_webhook_config not registered")
	}
	if names := schemaRequired(t, set); contains(names, "secret") {
		t.Errorf("set_org_webhook_config must not require secret, got %v", names)
	}
}
