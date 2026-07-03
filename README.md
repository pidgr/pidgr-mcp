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
      "command": "pidgr-mcp"
    }
  }
}
```

No keys to configure. The first time the server runs it opens your browser to
authenticate with Pidgr (passkey) and approve the requested scopes. The
resulting tokens are cached in your OS keychain — see [Authentication](#authentication).

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

Your MCP client authenticates over OAuth and presents a Pidgr bearer token on
each request — see [Authentication](#authentication).

### Docker

```bash
docker run -e PIDGR_MCP_TRANSPORT=http -p 8080:8080 ghcr.io/pidgr/pidgr-mcp:latest
```

## Authentication

`pidgr-mcp` authenticates with Pidgr over OAuth 2.1. There are no API keys —
authentication is OAuth-only in both transports.

### stdio (local)

Browser-based OAuth using the authorization-code flow with PKCE and an RFC 8252
loopback redirect:

1. On the first request the server opens your browser to the Pidgr login page.
2. You sign in (passkey) and approve the requested scopes for your user or
   organization.
3. The server captures the authorization code on a local loopback listener and
   exchanges it for tokens.
4. Tokens are cached in your OS keychain (with a `~/.config/pidgr/` file
   fallback, mode `0600`) and refreshed automatically. When a refresh fails the
   browser flow runs again.
5. Some sensitive operations require a recent authentication (OAuth step-up,
   RFC 9470). When the API answers with an `insufficient_user_authentication`
   challenge, the server re-opens your browser with `max_age=0` to force a
   fresh login, then retries the request once with the new token. If the retry
   is challenged again, the error is surfaced instead of looping.

The authorize and token endpoints are discovered from the issuer's
`/.well-known/oauth-authorization-server` document — nothing is hardcoded.

### HTTP (hosted)

The server runs as an OAuth resource server. Clients present a Pidgr OAuth
bearer token in the `Authorization` header; the token is verified against the
provider's JWKS and forwarded to the Pidgr API, which authorizes the request.
The server publishes `/.well-known/oauth-protected-resource` (RFC 9728) so MCP
clients can discover where to authenticate.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PIDGR_MCP_TRANSPORT` | No | `stdio` | `stdio` or `http` |
| `PIDGR_API_URL` | No | `https://api.pidgr.com` | API endpoint (gRPC base URL) |
| `PIDGR_INTEGRATIONS_URL` | No | `PIDGR_API_URL` | IntegrationsService endpoint (gRPC base URL). Falls back to `PIDGR_API_URL` when unset — useful when the IntegrationsService is co-hosted at the same gRPC ingress as the main API. |
| `PIDGR_MCP_ADDR` | No | `:8080` | Listen address (http mode) |
| `PIDGR_OAUTH_ISSUER` | No | `https://auth.pidgr.com` | OAuth issuer URL (authorization server) |
| `PIDGR_OAUTH_CLIENT_ID` | No | `pidgr-mcp` | OAuth client id |
| `PIDGR_OAUTH_SCOPE` | No | the Pidgr scope set | Space-separated scopes requested at authorization |
| `PIDGR_MCP_RESOURCE_URL` | No | `https://mcp.pidgr.com` | Resource identifier advertised in protected-resource metadata (http mode) |

## License

Apache 2.0
