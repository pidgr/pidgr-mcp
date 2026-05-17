// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pidgr/pidgr-mcp/internal/transport"
)

// RegisterAll registers all MCP tools on the server.
func RegisterAll(s *mcp.Server, c *transport.Clients) {
	registerCampaignTools(s, c)
	registerTemplateTools(s, c)
	registerGroupTools(s, c)
	registerTeamTools(s, c)
	registerMemberTools(s, c)
	registerOrganizationTools(s, c)
	registerRoleTools(s, c)
	registerApiKeyTools(s, c)
	registerHeatmapTools(s, c)
	registerReplayTools(s, c)
	registerPrivacyTools(s, c)
	registerAuditTools(s, c)
	registerSSOTools(s, c)
	registerInviteLinkTools(s, c)
	registerDeviceTools(s, c)
	registerInsightsTools(s, c)
	registerTemplateTranslationTools(s, c)
	registerIntegrationsTools(s, c)
}
