# pidgr-mcp

MCP (Model Context Protocol) server for Pidgr. Translates AI agent tool calls to API RPCs.

## Project Structure

```
cmd/pidgr-mcp/main.go      # Entrypoint: config, transport selection, auth wiring
internal/
  auth/                     # JWT verifier + Protected Resource Metadata
  transport/                # Client factory (static + dynamic token)
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
PIDGR_MCP_TRANSPORT=http PIDGR_AUTH_ISSUER=<issuer-url> go run ./cmd/pidgr-mcp/
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PIDGR_API_KEY` | stdio only | Scoped API key |
| `PIDGR_API_URL` | No | API endpoint |
| `PIDGR_MCP_TRANSPORT` | No | `stdio` or `http` |
| `PIDGR_MCP_ADDR` | No | Listen address (http) |
| `PIDGR_AUTH_ISSUER` | http only | OIDC issuer URL |
| `PIDGR_AUTH_CLIENT_ID` | No | App client ID for audience validation |

## OpenSpec

Changes for this repo are tracked in pidgr-admin's OpenSpec: `openspec/changes/mcp-server/`.
