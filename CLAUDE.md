# pidgr-mcp

MCP (Model Context Protocol) server for Pidgr. Translates AI agent tool calls to gRPC RPCs against pidgr-api.

## Tech Stack

- **Language**: Go
- **MCP SDK**: `github.com/modelcontextprotocol/go-sdk` (official)
- **gRPC Client**: Connect-Go with `connect.WithGRPC()` (uses `pidgrv1connect` stubs from pidgr-proto)
- **Auth**: Cognito JWT validation (HTTP mode) / API key forwarding (stdio mode)
- **License**: Apache 2.0

## Project Structure

```
cmd/pidgr-mcp/main.go      # Entrypoint: config, transport selection, auth wiring
internal/
  auth/                     # Cognito JWT verifier + Protected Resource Metadata
  transport/                # Connect-Go client factory (static + dynamic token)
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

# Run (stdio mode)
PIDGR_API_KEY=pidgr_k_... PIDGR_API_URL=http://localhost:50051 go run ./cmd/pidgr-mcp/

# Run (HTTP mode)
PIDGR_MCP_TRANSPORT=http PIDGR_COGNITO_POOL_ID=us-east-1_xxx go run ./cmd/pidgr-mcp/
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PIDGR_API_KEY` | stdio only | — | Scoped API key |
| `PIDGR_API_URL` | No | `https://api.pidgr.com` | gRPC endpoint |
| `PIDGR_MCP_TRANSPORT` | No | `stdio` | `stdio` or `http` |
| `PIDGR_MCP_ADDR` | No | `:8080` | Listen address (http) |
| `PIDGR_COGNITO_POOL_ID` | http only | — | Cognito User Pool ID |
| `PIDGR_COGNITO_REGION` | http only | `us-east-1` | Cognito region |

## Tools (49)

| Service | Count | Tools |
|---------|-------|-------|
| Campaign | 7 | create/update/start/get/list/cancel_campaign, list_deliveries |
| Template | 4 | create/update/get/list_template(s) |
| Group | 9 | create/get/list/update/delete_group, add/remove/list_group_members, get_user_group_memberships |
| Team | 8 | create/get/list/update/delete_team, add/remove/list_team_members |
| Member | 6 | invite/get/list_user(s), update_user_role, deactivate_user, update_user_profile |
| Organization | 4 | create/get/update_organization, update_sso_attribute_mappings |
| Role | 4 | list/create/update/delete_role(s) |
| ApiKey | 3 | create/list/revoke_api_key(s) |
| Heatmap | 2 | query_heatmap_data, list_screenshots |
| Replay | 2 | list_session_recordings, get_session_snapshots |

## Key Patterns

- All tool handlers follow the same pattern: build proto request → call Connect client → return `ProtoResult` or `ErrorResult`
- Responses are serialized with `protojson.Marshal(EmitUnpopulated: false)` for concise AI-readable JSON
- Enum inputs accept both full names (`DELIVERY_STATUS_PENDING`) and short names (`PENDING`)
- Token forwarding: stdio injects static API key, HTTP injects per-request JWT from MCP auth context

## OpenSpec

Changes for this repo are tracked in pidgr-admin's OpenSpec: `openspec/changes/mcp-server/`.
