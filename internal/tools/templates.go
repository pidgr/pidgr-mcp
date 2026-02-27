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

type TemplateVariableInput struct {
	Name         string `json:"name" jsonschema:"Variable name used in template body"`
	Description  string `json:"description,omitempty" jsonschema:"Human-readable description"`
	Required     bool   `json:"required,omitempty" jsonschema:"Whether this variable must be provided during rendering"`
	Source       string `json:"source,omitempty" jsonschema:"Value source: PROFILE or CUSTOM"`
	DefaultValue string `json:"default_value,omitempty" jsonschema:"Fallback value when source does not provide one"`
}

type CreateTemplateInput struct {
	Name      string                  `json:"name" jsonschema:"Template name (max 200 chars)"`
	Body      string                  `json:"body" jsonschema:"Template body with {{variable}} placeholders (max 50000 chars)"`
	Title     string                  `json:"title" jsonschema:"User-facing title shown as message subject (max 200 chars)"`
	Variables []TemplateVariableInput `json:"variables,omitempty" jsonschema:"Variables available for substitution"`
	Type      string                  `json:"type,omitempty" jsonschema:"Content format: MARKDOWN (default), RICH, or HTML"`
}

type UpdateTemplateInput struct {
	TemplateID string                  `json:"template_id" jsonschema:"Template UUID to update"`
	Body       string                  `json:"body" jsonschema:"New template body (max 50000 chars)"`
	Variables  []TemplateVariableInput `json:"variables,omitempty" jsonschema:"Updated variables"`
}

type GetTemplateInput struct {
	TemplateID string `json:"template_id" jsonschema:"Template UUID to retrieve"`
	Version    int32  `json:"version,omitempty" jsonschema:"Version to retrieve (0 = latest)"`
}

type ListTemplatesInput struct {
	PageSize  int32  `json:"page_size,omitempty" jsonschema:"Max items per page"`
	PageToken string `json:"page_token,omitempty" jsonschema:"Pagination token from previous response"`
	Type      string `json:"type,omitempty" jsonschema:"Filter by template type: MARKDOWN, RICH, or HTML"`
}

// ── Registration ────────────────────────────────────────────────────────────

func toProtoVariables(vars []TemplateVariableInput) []*pidgrv1.TemplateVariable {
	result := make([]*pidgrv1.TemplateVariable, len(vars))
	for i, v := range vars {
		source := pidgrv1.TemplateVariableSource_TEMPLATE_VARIABLE_SOURCE_UNSPECIFIED
		if s, ok := pidgrv1.TemplateVariableSource_value[v.Source]; ok {
			source = pidgrv1.TemplateVariableSource(s)
		} else if s, ok := pidgrv1.TemplateVariableSource_value["TEMPLATE_VARIABLE_SOURCE_"+v.Source]; ok {
			source = pidgrv1.TemplateVariableSource(s)
		}
		result[i] = &pidgrv1.TemplateVariable{
			Name:         v.Name,
			Description:  v.Description,
			Required:     v.Required,
			Source:       source,
			DefaultValue: v.DefaultValue,
		}
	}
	return result
}

func registerTemplateTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_template",
		Description: "Create a new versioned message template.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateTemplateInput) (*mcp.CallToolResult, any, error) {
		templateType := pidgrv1.TemplateType_TEMPLATE_TYPE_UNSPECIFIED
		if t, ok := pidgrv1.TemplateType_value[input.Type]; ok {
			templateType = pidgrv1.TemplateType(t)
		} else if t, ok := pidgrv1.TemplateType_value["TEMPLATE_TYPE_"+input.Type]; ok {
			templateType = pidgrv1.TemplateType(t)
		}
		resp, err := c.Templates.CreateTemplate(ctx, connect.NewRequest(&pidgrv1.CreateTemplateRequest{
			Name:      input.Name,
			Body:      input.Body,
			Variables: toProtoVariables(input.Variables),
			Title:     input.Title,
			Type:      templateType,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_template",
		Description: "Update a template, creating a new version.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTemplateInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Templates.UpdateTemplate(ctx, connect.NewRequest(&pidgrv1.UpdateTemplateRequest{
			TemplateId: input.TemplateID,
			Body:       input.Body,
			Variables:  toProtoVariables(input.Variables),
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_template",
		Description: "Retrieve a specific template by ID and optional version.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetTemplateInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Templates.GetTemplate(ctx, connect.NewRequest(&pidgrv1.GetTemplateRequest{
			TemplateId: input.TemplateID,
			Version:    input.Version,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_templates",
		Description: "List all templates for the organization with pagination.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTemplatesInput) (*mcp.CallToolResult, any, error) {
		templateType := pidgrv1.TemplateType_TEMPLATE_TYPE_UNSPECIFIED
		if t, ok := pidgrv1.TemplateType_value[input.Type]; ok {
			templateType = pidgrv1.TemplateType(t)
		} else if t, ok := pidgrv1.TemplateType_value["TEMPLATE_TYPE_"+input.Type]; ok {
			templateType = pidgrv1.TemplateType(t)
		}
		resp, err := c.Templates.ListTemplates(ctx, connect.NewRequest(&pidgrv1.ListTemplatesRequest{
			Pagination: &pidgrv1.Pagination{
				PageSize:  clampPageSize(input.PageSize),
				PageToken: input.PageToken,
			},
			Type: templateType,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
