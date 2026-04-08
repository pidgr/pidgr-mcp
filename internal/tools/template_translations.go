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

type CreateTemplateTranslationInput struct {
	TemplateID   string `json:"template_id" jsonschema:"Template UUID to translate"`
	Version      int32  `json:"version" jsonschema:"Template version to translate"`
	Locale       string `json:"locale" jsonschema:"Target locale (e.g., es, pt-BR, zh, ja)"`
	Title        string `json:"title" jsonschema:"Translated title"`
	Body         string `json:"body" jsonschema:"Translated body content with {{variable}} placeholders preserved"`
	TranslatedBy string `json:"translated_by" jsonschema:"Who created this translation (user UUID or 'ai:bedrock')"`
	Status       string `json:"status,omitempty" jsonschema:"Initial status: DRAFT or AI_TRANSLATED"`
}

type ListTemplateTranslationsInput struct {
	TemplateID string `json:"template_id" jsonschema:"Template UUID"`
	Version    int32  `json:"version,omitempty" jsonschema:"Template version (0 = latest)"`
}

type UpdateTemplateTranslationInput struct {
	TranslationID string `json:"translation_id" jsonschema:"Translation UUID to update"`
	Title         string `json:"title,omitempty" jsonschema:"Updated translated title (empty leaves unchanged)"`
	Body          string `json:"body,omitempty" jsonschema:"Updated translated body with {{variable}} placeholders preserved (empty leaves unchanged)"`
	Status        string `json:"status,omitempty" jsonschema:"Updated status: DRAFT, AI_TRANSLATED, IN_REVIEW, or APPROVED"`
}

type ApproveTemplateTranslationInput struct {
	TranslationID string `json:"translation_id" jsonschema:"Translation UUID to approve"`
}

// ── Registration ────────────────────────────────────────────────────────────

func registerTemplateTranslationTools(s *mcp.Server, c *transport.Clients) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_template_translation",
		Description: "Create a locale-specific translation of a template. Use list_template_translations first to check if a translation already exists for the locale.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateTemplateTranslationInput) (*mcp.CallToolResult, any, error) {
		status := pidgrv1.TranslationStatus_TRANSLATION_STATUS_DRAFT
		if s, ok := pidgrv1.TranslationStatus_value[input.Status]; ok {
			status = pidgrv1.TranslationStatus(s)
		} else if s, ok := pidgrv1.TranslationStatus_value["TRANSLATION_STATUS_"+input.Status]; ok {
			status = pidgrv1.TranslationStatus(s)
		}
		resp, err := c.Templates.CreateTemplateTranslation(ctx, connect.NewRequest(&pidgrv1.CreateTemplateTranslationRequest{
			TemplateId:   input.TemplateID,
			Version:      input.Version,
			Locale:       input.Locale,
			Title:        input.Title,
			Body:         input.Body,
			TranslatedBy: input.TranslatedBy,
			Status:       status,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_template_translations",
		Description: "List all translations for a template version. Shows status of each locale (DRAFT, AI_TRANSLATED, IN_REVIEW, APPROVED).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTemplateTranslationsInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Templates.ListTemplateTranslations(ctx, connect.NewRequest(&pidgrv1.ListTemplateTranslationsRequest{
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
		Name:        "update_template_translation",
		Description: "Update a template translation's title, body, or status. Use list_template_translations first to find translation UUIDs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateTemplateTranslationInput) (*mcp.CallToolResult, any, error) {
		status := pidgrv1.TranslationStatus_TRANSLATION_STATUS_UNSPECIFIED
		if s, ok := pidgrv1.TranslationStatus_value[input.Status]; ok {
			status = pidgrv1.TranslationStatus(s)
		} else if s, ok := pidgrv1.TranslationStatus_value["TRANSLATION_STATUS_"+input.Status]; ok {
			status = pidgrv1.TranslationStatus(s)
		}
		resp, err := c.Templates.UpdateTemplateTranslation(ctx, connect.NewRequest(&pidgrv1.UpdateTemplateTranslationRequest{
			TranslationId: input.TranslationID,
			Title:         input.Title,
			Body:          input.Body,
			Status:        status,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "approve_template_translation",
		Description: "Approve a template translation for use in campaigns. Requires TEMPLATES_REVIEW permission.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ApproveTemplateTranslationInput) (*mcp.CallToolResult, any, error) {
		resp, err := c.Templates.ApproveTemplateTranslation(ctx, connect.NewRequest(&pidgrv1.ApproveTemplateTranslationRequest{
			TranslationId: input.TranslationID,
		}))
		if err != nil {
			r, _ := convert.ErrorResult(err)
			return r, nil, nil
		}
		r, err := convert.ProtoResult(resp.Msg)
		return r, nil, err
	})
}
