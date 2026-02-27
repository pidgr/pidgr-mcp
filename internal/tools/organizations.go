// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"context"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	pidgrv1 "github.com/pidgr/pidgr-proto/gen/go/pidgr/v1"
	"github.com/pidgr/pidgr-mcp/internal/convert"
	"github.com/pidgr/pidgr-mcp/internal/transport"
)

// ── Input types ─────────────────────────────────────────────────────────────

type CreateOrganizationInput struct {
	Name        string `json:"name" jsonschema:"Organization name (max 200 chars)"`
	AdminEmail  string `json:"admin_email,omitempty" jsonschema:"Email for the initial admin user (required for API key auth)"`
	Industry    string `json:"industry,omitempty" jsonschema:"Industry: TECHNOLOGY/FINANCE/HEALTHCARE/EDUCATION/RETAIL/MANUFACTURING/MEDIA/OTHER"`
	CompanySize string `json:"company_size,omitempty" jsonschema:"Employee count: 1_200/200_500/500_1000/1000_5000/5000_PLUS"`
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

type UpdateSsoAttributeMappingsInput struct {
	SsoAttributeMappings []SsoMappingInput `json:"sso_attribute_mappings" jsonschema:"Complete list of SSO mappings (replaces all existing)"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerOrganizationTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_organization",
		Description: "Create a new organization with an initial admin user.",
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
			Name:        input.Name,
			AdminEmail:  input.AdminEmail,
			Industry:    industry,
			CompanySize: companySize,
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
}
