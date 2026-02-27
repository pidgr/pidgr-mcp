# pidgr-mcp

Open-source [MCP](https://modelcontextprotocol.io/) server for [Pidgr](https://pidgr.com).

Pidgr is an internal communication platform that replaces passive email and chat announcements with structured, trackable campaigns. Messages reach every employee, actions are verified, and delivery is measurable — not buried in a feed.

`pidgr-mcp` lets AI agents manage Pidgr through natural language. It exposes 49 tools and works with Claude Code, Cursor, Windsurf, and any MCP-compatible client.

## Capabilities

**Campaigns** — Create, update, start, cancel, and list campaigns. Track per-user delivery status (sent, delivered, acknowledged, missed).

**Templates** — Create versioned message templates with variable substitution. Supports Markdown, Rich, and HTML content types.

**Audience** — Manage recipient groups and organizational teams. Add/remove members, query memberships in batch.

**Users** — Invite users, manage profiles (department, title, location), assign roles, deactivate accounts.

**Organizations** — Configure organization settings, default workflows, industry, and SSO attribute mappings.

**Roles & Permissions** — Create custom roles with granular permission sets. Assign roles to users.

**API Keys** — Create scoped API keys with optional expiration. List and revoke keys.

**Analytics** — Query aggregated touch heatmap data with screen, campaign, and time range filters. List session recordings and fetch snapshot data for playback.

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
docker run -e PIDGR_MCP_TRANSPORT=http -e PIDGR_AUTH_POOL_ID=<your-pool-id> -p 8080:8080 ghcr.io/pidgr/pidgr-mcp:latest
```

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `PIDGR_API_KEY` | stdio only | Scoped API key |
| `PIDGR_API_URL` | No | API endpoint |
| `PIDGR_MCP_TRANSPORT` | No | `stdio` or `http` |
| `PIDGR_MCP_ADDR` | No | Listen address (http mode) |
| `PIDGR_AUTH_POOL_ID` | http only | Auth provider pool ID |
| `PIDGR_AUTH_REGION` | http only | Auth provider region |
| `PIDGR_AUTH_CLIENT_ID` | No | App client ID for audience validation |

## License

Apache 2.0
