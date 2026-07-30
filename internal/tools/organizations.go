// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"github.com/pidgr/pidgr-mcp/internal/convert"
	"github.com/pidgr/pidgr-mcp/internal/transport"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── Input types ─────────────────────────────────────────────────────────────

type ListUserOrganizationsInput struct{}

type CreateOrganizationInput struct {
	Name                 string `json:"name" jsonschema:"Organization name (max 200 chars)"`
	Industry             string `json:"industry,omitempty" jsonschema:"Industry: TECHNOLOGY/FINANCE/HEALTHCARE/EDUCATION/RETAIL/MANUFACTURING/MEDIA/OTHER"`
	CompanySize          string `json:"company_size,omitempty" jsonschema:"Employee count: 1_200/200_500/500_1000/1000_5000/5000_PLUS"`
	DataGovernanceRegion string `json:"data_governance_region,omitempty" jsonschema:"Data governance framework: EU, LATAM, APAC, US (default: US)"`
	FixtureID            string `json:"fixture_id,omitempty" jsonschema:"Bootstrap fixture id from list_sandbox_fixtures (e.g. starter/empty/fintech/sales). Empty applies the default fixture."`
}

type ListUserSandboxesInput struct{}

type ListSandboxFixturesInput struct{}

type CreateSandboxOrganizationInput struct {
	Name                 string `json:"name" jsonschema:"Sandbox organization name"`
	ExpiresInDays        int32  `json:"expires_in_days" jsonschema:"TTL in days (max 30, max 14 with SCIM)"`
	DataGovernanceRegion string `json:"data_governance_region,omitempty" jsonschema:"Data governance: EU, LATAM, APAC, US (default: US)"`
	FixtureID            string `json:"fixture_id,omitempty" jsonschema:"Bootstrap fixture id from list_sandbox_fixtures (e.g. starter/empty/fintech/sales). Empty applies the default fixture."`
}

type GetOrganizationInput struct{}

type UpdateOrganizationInput struct {
	Name            string                      `json:"name,omitempty" jsonschema:"New organization name"`
	DefaultWorkflow *pidgrv1.WorkflowDefinition `json:"default_workflow,omitempty" jsonschema:"New default workflow DAG"`
	Industry        string                      `json:"industry,omitempty" jsonschema:"New industry"`
	CompanySize     string                      `json:"company_size,omitempty" jsonschema:"New company size"`
}

type SsoMappingInput struct {
	IdpClaim     string `json:"idp_claim" jsonschema:"Claim name from identity provider"`
	ProfileField string `json:"profile_field" jsonschema:"Target profile field name"`
}

type RotateAnalyticsSaltInput struct {
	NewBucketCount int32 `json:"new_bucket_count,omitempty" jsonschema:"New bucket count (must be >= current, 0 keeps current)"`
}

type UpdateAnalyticsEpsilonInput struct {
	Epsilon float32 `json:"epsilon" jsonschema:"Differential privacy epsilon (0.5 to 5.0)"`
}

type GetOrgPrivacySettingsInput struct{}

// UpdateOrgPrivacySettingsInput mirrors the proto's three optional bools:
// every toggle is tri-state, so a nil pointer leaves that setting untouched
// and a caller can flip one without restating the other two.
type UpdateOrgPrivacySettingsInput struct {
	AiClusteringEnabled        *bool `json:"ai_clustering_enabled,omitempty" jsonschema:"Enable or disable ML archetype clustering and ACK predictions. Omit to leave unchanged"`
	BehavioralAnalyticsEnabled *bool `json:"behavioral_analytics_enabled,omitempty" jsonschema:"Enable or disable behavioral analytics. While disabled, tap-event ingestion is accepted and silently dropped and the Compass tap heatmap stays empty. Omit to leave unchanged"`
	ThirdPartyChannelsEnabled  *bool `json:"third_party_channels_enabled,omitempty" jsonschema:"Enable or disable third-party notification channels. Omit to leave unchanged"`
}

type DeleteSandboxOrganizationInput struct {
	OrgID string `json:"org_id" jsonschema:"UUID of the sandbox organization to delete. Verify it with list_user_sandboxes first"`
}

type UpdateSsoAttributeMappingsInput struct {
	SsoAttributeMappings []SsoMappingInput `json:"sso_attribute_mappings" jsonschema:"Complete list of SSO mappings (replaces all existing)"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerOrganizationTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_organization",
		Description: "Create a new organization. Requires JWT auth (HTTP mode with user JWT) — the caller becomes the initial admin. Stdio + static API key callers are rejected; use invite links to add further admins after creation.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateOrganizationInput) (*mcp.CallToolResult, any, error) {
		industry := pidgrv1.Industry_INDUSTRY_UNSPECIFIED
		if v, ok := pidgrv1.Industry_value[input.Industry]; ok {
			industry = pidgrv1.Industry(v)
		} else if v, ok := pidgrv1.Industry_value["INDUSTRY_"+input.Industry]; ok {
			industry = pidgrv1.Industry(v)
		}
		companySize := pidgrv1.CompanySize_COMPANY_SIZE_UNSPECIFIED
		if v, ok := pidgrv1.CompanySize_value[input.CompanySize]; ok {
			companySize = pidgrv1.CompanySize(v)
		} else if v, ok := pidgrv1.CompanySize_value["COMPANY_SIZE_"+input.CompanySize]; ok {
			companySize = pidgrv1.CompanySize(v)
		}
		resp, err := c.Organizations.CreateOrganization(ctx, connect.NewRequest(&pidgrv1.CreateOrganizationRequest{
			Name:                 input.Name,
			Industry:             industry,
			CompanySize:          companySize,
			DataGovernanceRegion: input.DataGovernanceRegion,
			FixtureId:            input.FixtureID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_organization",
		Description: "Retrieve the organization for the authenticated user.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetOrganizationInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.GetOrganization(ctx, connect.NewRequest(&pidgrv1.GetOrganizationRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_organization",
		Description: "Update organization settings.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateOrganizationInput) (*mcp.CallToolResult, any, error) {
		industry := pidgrv1.Industry_INDUSTRY_UNSPECIFIED
		if v, ok := pidgrv1.Industry_value[input.Industry]; ok {
			industry = pidgrv1.Industry(v)
		} else if v, ok := pidgrv1.Industry_value["INDUSTRY_"+input.Industry]; ok {
			industry = pidgrv1.Industry(v)
		}
		companySize := pidgrv1.CompanySize_COMPANY_SIZE_UNSPECIFIED
		if v, ok := pidgrv1.CompanySize_value[input.CompanySize]; ok {
			companySize = pidgrv1.CompanySize(v)
		} else if v, ok := pidgrv1.CompanySize_value["COMPANY_SIZE_"+input.CompanySize]; ok {
			companySize = pidgrv1.CompanySize(v)
		}
		resp, err := c.Organizations.UpdateOrganization(ctx, connect.NewRequest(&pidgrv1.UpdateOrganizationRequest{
			Name:            input.Name,
			DefaultWorkflow: input.DefaultWorkflow,
			Industry:        industry,
			CompanySize:     companySize,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_sso_attribute_mappings",
		Description: "Replace all SSO identity provider claim-to-profile field mappings.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateSsoAttributeMappingsInput) (*mcp.CallToolResult, any, error) {
		mappings := make([]*pidgrv1.SsoAttributeMapping, len(input.SsoAttributeMappings))
		for i, m := range input.SsoAttributeMappings {
			mappings[i] = &pidgrv1.SsoAttributeMapping{
				IdpClaim:     m.IdpClaim,
				ProfileField: m.ProfileField,
			}
		}
		resp, err := c.Organizations.UpdateSsoAttributeMappings(ctx, connect.NewRequest(&pidgrv1.UpdateSsoAttributeMappingsRequest{
			SsoAttributeMappings: mappings,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rotate_analytics_salt",
		Description: "Rotate the k-anonymization salt and optionally increase the bucket count. Existing data keeps old bucket assignments.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input RotateAnalyticsSaltInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.RotateAnalyticsSalt(ctx, connect.NewRequest(&pidgrv1.RotateAnalyticsSaltRequest{
			NewBucketCount: input.NewBucketCount,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_analytics_epsilon",
		Description: "Update the differential privacy epsilon parameter (0.5 to 5.0). Lower = more privacy, higher = more precision.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateAnalyticsEpsilonInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.UpdateAnalyticsEpsilon(ctx, connect.NewRequest(&pidgrv1.UpdateAnalyticsEpsilonRequest{
			Epsilon: input.Epsilon,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_user_organizations",
		Description: "List all organizations the authenticated user belongs to. Excludes expired sandbox orgs. No org context required.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListUserOrganizationsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.ListUserOrganizations(ctx, connect.NewRequest(&pidgrv1.ListUserOrganizationsRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_user_sandboxes",
		Description: "List the authenticated user's sandbox organizations (org_type=SANDBOX), ordered by expires_at ascending (soonest-expiring first). Excludes already-expired sandboxes. User-scoped; no org context required.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListUserSandboxesInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.ListUserSandboxes(ctx, connect.NewRequest(&pidgrv1.ListUserSandboxesRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_sandbox_fixtures",
		Description: "List the bootstrap fixture catalog (e.g. starter/empty/fintech/sales) available for seeding new organizations and sandboxes. Each entry has an id, name, description, and is_default flag. Pass the chosen id as fixture_id to create_organization or create_sandbox_organization. No org context required.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListSandboxFixturesInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.ListSandboxFixtures(ctx, connect.NewRequest(&pidgrv1.ListSandboxFixturesRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_sandbox_organization",
		Description: "Create a sandbox organization for testing configurations. Auto-deletes after TTL. SCIM provisioning allowed for IdP testing (DB-only, no Cognito users).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateSandboxOrganizationInput) (*mcp.CallToolResult, any, error) {
		expiresAt := timestamppb.New(time.Now().Add(time.Duration(input.ExpiresInDays) * 24 * time.Hour))
		resp, err := c.Organizations.CreateSandboxOrganization(ctx, connect.NewRequest(&pidgrv1.CreateSandboxOrganizationRequest{
			Name:                 input.Name,
			ExpiresAt:            expiresAt,
			DataGovernanceRegion: input.DataGovernanceRegion,
			FixtureId:            input.FixtureID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "delete_sandbox_organization",
		Description: "PERMANENTLY delete a sandbox organization and everything in it. Sandbox-only: standard organizations are rejected. " +
			"Irreversible — resolve the id with list_user_sandboxes and confirm the name before calling.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteSandboxOrganizationInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.DeleteSandboxOrganization(ctx, connect.NewRequest(&pidgrv1.DeleteSandboxOrganizationRequest{
			OrgId: input.OrgID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_org_privacy_settings",
		Description: "Read the org-level privacy toggles (AI clustering, behavioral analytics, third-party channels) with the consent-trace metadata for each: who last changed it and when. " +
			"Start here when an analytics or clustering surface is unexpectedly empty — a disabled toggle degrades silently rather than erroring.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetOrgPrivacySettingsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.GetOrgPrivacySettings(ctx, connect.NewRequest(&pidgrv1.GetOrgPrivacySettingsRequest{}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "update_org_privacy_settings",
		Description: "Flip the org-level privacy toggles. Each is tri-state: omit a field to leave it unchanged. " +
			"behavioral_analytics_enabled gates tap-event ingestion and the Compass tap heatmap; ai_clustering_enabled gates archetype clustering and ACK predictions; third_party_channels_enabled gates non-push channels. Every change is recorded with the acting user for the consent trace.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateOrgPrivacySettingsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Organizations.UpdateOrgPrivacySettings(ctx, connect.NewRequest(&pidgrv1.UpdateOrgPrivacySettingsRequest{
			AiClusteringEnabled:        input.AiClusteringEnabled,
			BehavioralAnalyticsEnabled: input.BehavioralAnalyticsEnabled,
			ThirdPartyChannelsEnabled:  input.ThirdPartyChannelsEnabled,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
