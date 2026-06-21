# pidgr-mcp

MCP (Model Context Protocol) server for Pidgr. Translates AI agent tool calls to API RPCs.

## Project Structure

```
cmd/pidgr-mcp/main.go      # Entrypoint: config, transport selection, auth wiring
internal/
  auth/                     # API key verifier
  oauth/                    # stdio OAuth client (authorization_code+PKCE+loopback)
  transport/                # Client factory (static + dynamic + OAuth token)
  tools/                    # 49 MCP tools across 10 services
  convert/                  # ProtoResult, ErrorResult, SuccessResult helpers
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

# Run (stdio mode)
PIDGR_API_KEY=pidgr_k_... PIDGR_API_URL=http://localhost:50051 go run ./cmd/pidgr-mcp/

# Run (HTTP mode)
PIDGR_MCP_TRANSPORT=http go run ./cmd/pidgr-mcp/
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PIDGR_API_KEY` | No | Scoped `pidgr_k_` key. When set, takes precedence over OAuth (backward compat). |
| `PIDGR_API_URL` | No | API endpoint |
| `PIDGR_MCP_TRANSPORT` | No | `stdio` or `http` |
| `PIDGR_MCP_ADDR` | No | Listen address (http) |
| `PIDGR_OAUTH_ISSUER` | No | OAuth issuer base URL (default `https://auth.pidgr.com`). Discovery from `{issuer}/.well-known/oauth-authorization-server`. |
| `PIDGR_OAUTH_CLIENT_ID` | No | Static public client ID (default `pidgr-mcp`). |
| `PIDGR_OAUTH_SCOPE` | No | Space-delimited scopes (default `openid offline_access`). |

### stdio authentication

stdio auth resolves in this order:

1. If `PIDGR_API_KEY` (a `pidgr_k_` key) is set, it is injected as the Bearer on
   every RPC (backward compatible).
2. Otherwise the **OAuth** authorization_code + PKCE + RFC 8252 loopback flow runs:
   discovery → browser consent on the issuer → loopback captures the code →
   `/token` exchange → access + rotating refresh tokens stored in the OS keychain
   (with a `~/.config/pidgr/oauth-token.json` 0600 file fallback). Tokens refresh
   automatically; if refresh fails the browser flow re-runs.

## OpenSpec

This repo does not carry its own `openspec/` change directory. New behavior is reviewed upstream against the proto and service contracts this server wraps.
