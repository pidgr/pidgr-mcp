# pidgr-mcp

MCP server for [Pidgr](https://pidgr.com) — exposes 49 tools covering campaigns, templates, groups, teams, members, organizations, roles, API keys, heatmaps, and session replay. Works with Claude Code, Cursor, Windsurf, and any MCP-compatible AI client.

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
