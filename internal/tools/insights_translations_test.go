// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import "testing"

// ─── Insights Tools ─────────────────────────────────────────────────────────

func TestInsightsToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{"get_group_archetypes", "predict_campaign_ack", "get_campaign_advisory"}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("insights tool %q not registered", name)
		}
	}
}

func TestGetGroupArchetypesRequiresGroupID(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "get_group_archetypes")
	if tool == nil {
		t.Fatal("get_group_archetypes not found")
	}
	if !schemaHasProperty(t, tool, "group_id") {
		t.Error("get_group_archetypes missing group_id property")
	}
}

func TestPredictCampaignACKRequiresGroupID(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "predict_campaign_ack")
	if tool == nil {
		t.Fatal("predict_campaign_ack not found")
	}
	if !schemaHasProperty(t, tool, "group_id") {
		t.Error("predict_campaign_ack missing group_id property")
	}
}

func TestGetCampaignAdvisoryHasOptionalFields(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "get_campaign_advisory")
	if tool == nil {
		t.Fatal("get_campaign_advisory not found")
	}
	if !schemaHasProperty(t, tool, "group_id") {
		t.Error("get_campaign_advisory missing group_id property")
	}
	if !schemaHasProperty(t, tool, "template_id") {
		t.Error("get_campaign_advisory missing template_id property")
	}
}

// ─── Template Translation Tools ─────────────────────────────────────────────

func TestTemplateTranslationToolsRegistered(t *testing.T) {
	tools := listTools(t)
	expected := []string{"create_template_translation", "list_template_translations", "approve_template_translation"}
	for _, name := range expected {
		if toolByName(tools, name) == nil {
			t.Errorf("translation tool %q not registered", name)
		}
	}
}

func TestCreateTemplateTranslationRequiresFields(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "create_template_translation")
	if tool == nil {
		t.Fatal("create_template_translation not found")
	}
	for _, prop := range []string{"template_id", "version", "locale", "title", "body", "translated_by"} {
		if !schemaHasProperty(t, tool, prop) {
			t.Errorf("create_template_translation missing %s property", prop)
		}
	}
}

func TestListTemplateTranslationsRequiresTemplateID(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "list_template_translations")
	if tool == nil {
		t.Fatal("list_template_translations not found")
	}
	if !schemaHasProperty(t, tool, "template_id") {
		t.Error("list_template_translations missing template_id property")
	}
}

func TestApproveTemplateTranslationRequiresTranslationID(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "approve_template_translation")
	if tool == nil {
		t.Fatal("approve_template_translation not found")
	}
	if !schemaHasProperty(t, tool, "translation_id") {
		t.Error("approve_template_translation missing translation_id property")
	}
}

// ─── Sandbox Organization Tool ──────────────────────────────────────────────

func TestCreateSandboxOrganizationRegistered(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "create_sandbox_organization")
	if tool == nil {
		t.Fatal("create_sandbox_organization not registered")
	}
}

func TestCreateSandboxOrganizationRequiresFields(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "create_sandbox_organization")
	if tool == nil {
		t.Fatal("create_sandbox_organization not found")
	}
	if !schemaHasProperty(t, tool, "name") {
		t.Error("create_sandbox_organization missing name property")
	}
	if !schemaHasProperty(t, tool, "expires_in_days") {
		t.Error("create_sandbox_organization missing expires_in_days property")
	}
}

// ─── Data Governance Region ─────────────────────────────────────────────────

func TestCreateOrganizationHasDataGovernanceRegion(t *testing.T) {
	tools := listTools(t)
	tool := toolByName(tools, "create_organization")
	if tool == nil {
		t.Fatal("create_organization not found")
	}
	if !schemaHasProperty(t, tool, "data_governance_region") {
		t.Error("create_organization missing data_governance_region property")
	}
}

// ─── Tool Descriptions ──────────────────────────────────────────────────────

func TestInsightsTranslationSandboxToolsHaveDescriptions(t *testing.T) {
	tools := listTools(t)
	bulbasaurTools := []string{
		"get_group_archetypes", "predict_campaign_ack", "get_campaign_advisory",
		"create_template_translation", "list_template_translations", "approve_template_translation",
		"create_sandbox_organization",
	}
	for _, name := range bulbasaurTools {
		tool := toolByName(tools, name)
		if tool == nil {
			t.Errorf("tool %q not found", name)
			continue
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", name)
		}
	}
}
