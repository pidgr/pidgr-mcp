# pidgr-mcp

MCP server for [Pidgr](https://pidgr.com) — translates AI agent tool calls to gRPC RPCs.

Exposes 49 tools covering campaigns, templates, groups, teams, members, organizations, roles, API keys, heatmaps, and session replay. Works with Claude Code, Cursor, Windsurf, and any MCP-compatible AI client.

## Install

### Binary (stdio)

Download from [GitHub Releases](https://github.com/pidgr/pidgr-mcp/releases) and add to your MCP config:

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
docker run -e PIDGR_MCP_TRANSPORT=http -e PIDGR_COGNITO_POOL_ID=us-east-1_xxx -p 8080:8080 ghcr.io/pidgr/pidgr-mcp:latest
```

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PIDGR_API_KEY` | stdio only | — | Scoped API key |
| `PIDGR_API_URL` | No | `https://api.pidgr.com` | gRPC endpoint |
| `PIDGR_MCP_TRANSPORT` | No | `stdio` | `stdio` or `http` |
| `PIDGR_MCP_ADDR` | No | `:8080` | Listen address (http) |
| `PIDGR_COGNITO_POOL_ID` | http only | — | Cognito User Pool ID |
| `PIDGR_COGNITO_REGION` | http only | `us-east-1` | Cognito region |

## License

Apache 2.0
