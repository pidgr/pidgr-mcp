# pidgr-mcp

Open-source [MCP](https://modelcontextprotocol.io/) server for [Pidgr](https://pidgr.com).

Pidgr is an internal communication platform that replaces passive email and chat announcements with structured, trackable campaigns. Messages reach every employee, actions are verified, and delivery is measurable — not buried in a feed.

`pidgr-mcp` lets AI agents manage Pidgr through natural language. It exposes 95 tools and works with Claude Code, Cursor, Windsurf, and any MCP-compatible client.

## Capabilities

**Campaigns** — Create, update, start, cancel, and list campaigns. Track per-user delivery status (sent, delivered, acknowledged, missed). Inspect the per-archetype tendency-shift breakdown after a campaign completes (including email-engagement fields: `email_delivered_count`, `email_open_rate_real`, `email_open_rate_raw`).

**Templates** — Create versioned message templates with variable substitution. Supports Markdown, Rich, and HTML content types.

**Audience** — Manage recipient groups and organizational teams. Add/remove members, query memberships in batch.

**Users** — Invite users, manage profiles (department, title, location), assign roles, deactivate accounts.

**Organizations** — Configure organization settings, default workflows, industry, and SSO attribute mappings.

**Roles & Permissions** — Create custom roles with granular permission sets. Assign roles to users.

**API Keys** — Create scoped API keys with optional expiration. List and revoke keys.

**Analytics** — Query aggregated touch heatmap data with screen, campaign, and time range filters. List session recordings and fetch snapshot data for playback.

**Integrations** — Manage per-channel recipient reachability (`list_reachabilities_for_user`, `upsert_reachability`, `remove_reachability`) and the per-`(org, channel)` cost-cap policy (`get_cost_cap_policy`, `set_cost_cap_policy`). Reachability identifiers (email addresses, phone numbers, Slack user ids, etc.) are sensitive — they are column-level KMS-encrypted server-side, never logged, and never returned over the wire. Calls route to the IntegrationsService endpoint (see `PIDGR_INTEGRATIONS_URL` below).

## Install

### Binary (stdio)

Download from [GitHub Releases](https://github.com/pidgr/pidgr-mcp/releases) and verify the checksum:

```bash
sha256sum -c checksums.txt
```

Add to your MCP config:

```json
{
  "mcpServers": {
    "pidgr": {
      "command": "pidgr-mcp",
      "env": { "PIDGR_API_KEY": "pidgr_k_..." }
    }
  }
}
```

### Hosted (Streamable HTTP)

```json
{
  "mcpServers": {
    "pidgr": {
      "url": "https://mcp.pidgr.com"
    }
  }
}
```

No API key needed — OAuth handles authentication.

### Docker

```bash
docker run -e PIDGR_MCP_TRANSPORT=http -e PIDGR_AUTH_ISSUER=<your-issuer-url> -p 8080:8080 ghcr.io/pidgr/pidgr-mcp:latest
```

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `PIDGR_API_KEY` | stdio only | Scoped API key |
| `PIDGR_API_URL` | No | API endpoint (gRPC base URL) |
| `PIDGR_INTEGRATIONS_URL` | No | IntegrationsService endpoint (gRPC base URL). Falls back to `PIDGR_API_URL` when unset — useful when the IntegrationsService is co-hosted at the same gRPC ingress as the main API. |
| `PIDGR_MCP_TRANSPORT` | No | `stdio` or `http` |
| `PIDGR_MCP_ADDR` | No | Listen address (http mode) |
| `PIDGR_AUTH_ISSUER` | http only | OIDC issuer URL |
| `PIDGR_AUTH_CLIENT_ID` | No | App client ID for audience validation |

## License

Apache 2.0
