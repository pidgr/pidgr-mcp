# pidgr-mcp

MCP (Model Context Protocol) server for Pidgr. Translates AI agent tool calls to API RPCs.

## Project Structure

```
cmd/pidgr-mcp/main.go      # Entrypoint: config, transport selection, auth wiring
internal/
  oauth/                    # stdio OAuth client (authorization_code+PKCE+loopback) + HTTP-mode bearer verifier
  transport/                # Client factory (dynamic context token + OAuth token)
  tools/                    # MCP tools across the Pidgr services
  convert/                  # ProtoResult, ErrorResult, SuccessResult helpers
  observability/            # OTEL tracer/logger init + slog fanout handler
```

## Development

```bash
# Build
GOPRIVATE=github.com/pidgr/* go build ./...

# Test
GOPRIVATE=github.com/pidgr/* go test ./... -cover

# Vet
go vet ./...

# Lint (includes gosec)
golangci-lint run

# Run (stdio mode) — opens the browser for OAuth on first use
PIDGR_API_URL=http://localhost:50051 go run ./cmd/pidgr-mcp/

# Run (HTTP mode)
PIDGR_MCP_TRANSPORT=http go run ./cmd/pidgr-mcp/
```

## Environment Variables

Authentication is OAuth-only in both transports — there are no API-key env vars.

| Variable | Required | Description |
|----------|----------|-------------|
| `PIDGR_API_URL` | No | API endpoint (default `https://api.pidgr.com`) |
| `PIDGR_INTEGRATIONS_URL` | No | IntegrationsService endpoint. Falls back to `PIDGR_API_URL` when unset. |
| `PIDGR_MCP_TRANSPORT` | No | `stdio` or `http` (default `stdio`) |
| `PIDGR_MCP_ADDR` | No | Listen address (http, default `:8080`) |
| `PIDGR_MCP_RESOURCE_URL` | No | Resource identifier advertised in protected-resource metadata (http, default `https://mcp.pidgr.com`) |
| `PIDGR_OAUTH_ISSUER` | No | OAuth issuer base URL (default `https://auth.pidgr.com`). Discovery from `{issuer}/.well-known/oauth-authorization-server`. |
| `PIDGR_OAUTH_CLIENT_ID` | No | Static public client ID (default `pidgr-mcp`). |
| `PIDGR_OAUTH_SCOPE` | No | Space-delimited scopes (default: the Pidgr scope set). |

### stdio authentication

stdio is OAuth-only. The authorization_code + PKCE + RFC 8252 loopback flow runs
on first use: discovery → browser consent on the issuer → loopback captures the
code → `/token` exchange → access + rotating refresh tokens stored in the OS
keychain (with a `~/.config/pidgr/oauth-token.json` 0600 file fallback). Tokens
refresh automatically; if refresh fails the browser flow re-runs.

### HTTP authentication

HTTP mode runs as an OAuth resource server. The bearer must be a Pidgr-issued
OAuth JWT, verified against the provider's JWKS and forwarded to the API for
authorization. The server serves `/.well-known/oauth-protected-resource` (RFC
9728) for discovery.

## OpenSpec

This repo does not carry its own `openspec/` change directory. New behavior is reviewed upstream against the proto and service contracts this server wraps.
