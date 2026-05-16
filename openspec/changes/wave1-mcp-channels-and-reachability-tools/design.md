## Context

pidgr-mcp is the open-source MCP server that translates AI-agent tool calls to pidgr-platform gRPC RPCs. v0.9.x exposes 49 tools across 10 services hosted by pidgr-api. Wave 1 splits a new RPC surface (`pidgr.v1.IntegrationsService`) onto a sibling service — `pidgr-integrations` — which owns the reachability registry, region policy, cost-cap state, and the dispatch worker for third-party channels. The same `pidgr-proto` package generates the Connect-Go client for both services, so the consumer-side wiring is identical to the existing 16 services on `transport.Clients` — just pointed at a (potentially) different base URL.

Wave 1 also added three Wave-1-specific fields to `pidgr.v1.ArchetypeShareShift` via the `add-email-open-rate-archetype-enrichment` change on pidgr-api: `email_delivered_count`, `email_open_rate_real`, `email_open_rate_raw`. The MCP tool surface does not include `GetCampaignArchetypeBreakdown` today, so the new fields are unreachable. This change exposes the RPC; the new fields then surface naturally via the generated proto type and `protojson.Marshal` in `convert.ProtoResult` — no field-level passthrough work.

Prerequisites:

- pidgr-integrations Wave 1 bootstrap (task 5.2) hosting `IntegrationsService` at a known endpoint.
- pidgr-proto v0.74.0+ on the consumed `go.mod` (already on `main` per `go.mod`).
- pidgr-api `add-email-open-rate-archetype-enrichment` shipped (already shipped; the proto fields are present in pidgr-proto v0.74.0).

## Goals / Non-Goals

**Goals:**

- AI agents can read + write a recipient's per-channel reachability identifiers via MCP.
- AI agents can read + write per-`(org, channel)` cost-cap policy via MCP.
- AI agents can read a campaign's archetype-shift breakdown including the Wave 1 email-open-rate fields.
- New tools follow the same 1:1 RPC-to-tool mapping pattern as the existing 49 tools (no aggregation, no client-side derived fields, no per-tool re-validation beyond what the upstream service enforces).

**Non-Goals:**

- Exposing `DispatchToChannel`. It's internal-mTLS-only by design (worker-mode entry point invoked by the Temporal worker); MCP callers should never invoke it.
- Exposing `GetReachability` (single-row lookup). `list_reachabilities_for_user` returns the same data shape for all of a user's channels in one call and is the only reachability-read surface admin agents need in Wave 1.
- Exposing `GetRegionPolicy` / `SetRegionPolicy`. Region policy is set once at provisioning time via admin UI in Wave 1; deferred to a follow-up change once we have a concrete agent-driven use case.
- Migrating the existing 49 tools' spec from pidgr-admin's OpenSpec to pidgr-mcp's own OpenSpec. That housekeeping move is its own change.
- Implementing the tools. This is a spec-only scaffold; the implementation PR depends on Wave 1 task 5.2 (pidgr-integrations bootstrap).

## Decisions

### 1. 1:1 tool ↔ RPC mapping for all six new tools

**Decision:** Each new MCP tool maps exactly one input struct → one Connect-Go RPC call → `convert.ProtoResult` of the proto response. No client-side merging, validation, or alternate-shape rewriting.

**Why:** This is the established pattern across all 49 existing tools (see `internal/tools/insights.go`, `campaigns.go`, etc.). Tool descriptions explain semantics; the response shape is whatever the proto generates. Any business-logic divergence would create a maintenance burden — agents and humans both read the proto definitions as the source of truth.

**Implication:** Server-side semantic quirks (e.g. `GetCostCapPolicy` returning the channel-default cap when no row exists; `RemoveReachability` returning `removed = false` rather than NOT_FOUND when the row is absent) are documented in the proto and surfaced verbatim in the tool description, not papered over with MCP-side logic.

### 2. `transport.Clients` gains an `Integrations` field

**Decision:** Add `Integrations pidgrv1connect.IntegrationsServiceClient` to `transport.Clients`. Construct it in `newClients` alongside the other 16 service clients with the same `connect.WithGRPC()` option and either the static or dynamic Bearer-token interceptor.

**Why not a second `Clients` struct keyed off `pidgr-integrations` baseURL:** The interceptor pattern is the same. The auth model is the same (Cognito JWT for the admin RPCs we're exposing; the internal-mTLS `DispatchToChannel` is not exposed). The only variable is the base URL.

**Base URL resolution:** Two options for the implementation PR:

1. Default to `PIDGR_INTEGRATIONS_URL` env var; fall back to `PIDGR_API_URL` when unset (works for the co-hosted-on-pidgr-api-ALB Wave 1 deployment).
2. Hard-require `PIDGR_INTEGRATIONS_URL` once pidgr-integrations is on `integrations.pidgr.com`.

The proposal defers this choice to the implementation PR because pidgr-integrations Wave 1 task 5.2 hasn't yet settled on a hostname. The spec only requires that the implementation read the value from env, not which env var.

**Open decision for user:** Confirm option 1 (fallback) vs option 2 (hard-require). The spec is compatible with both.

### 3. `upsert_reachability` flags `identifier_plaintext` as sensitive in the tool description

**Decision:** The MCP tool description SHALL include the line "the `identifier_plaintext` argument is sensitive (email address, phone number, etc.) — handle with care." The argument is passed unchanged to the upstream RPC; pidgr-mcp does NOT hash, mask, or log it on the client side. The upstream service is responsible for KMS encryption, lookup-hash computation, and ensuring the plaintext is never logged.

**Why:** Hashing client-side would create a second source-of-truth (which pepper? which version?). The whole point of the reachability registry's HMAC-pepper architecture is that hashing lives in one place — `pidgr-integrations`. pidgr-mcp's only job here is to relay the plaintext over the authenticated transport.

**Logging:** pidgr-mcp does NOT log RPC request bodies. The MCP SDK's tool-call telemetry SHALL be configured to redact `upsert_reachability` arguments when telemetry lands (currently no telemetry — observability work is tracked separately).

### 4. Channel argument uses `ChannelName` enum names, not numeric values

**Decision:** The `channel` argument on each tool is a string-typed JSON field accepting the canonical enum name (e.g. `"CHANNEL_NAME_EMAIL"`, `"CHANNEL_NAME_SLACK"`). Implementation uses `pidgrv1.ChannelName.Enum().Lookup(...)` (or equivalent generated helper) for parsing.

**Why:** Numeric enum values are fragile across proto regenerations (technically stable but visually confusing). Existing tools that take enum args (e.g. `list_deliveries.status_filter`, `query_heatmap_data.mode`) already use the name-string convention.

**Validation:** Out-of-range values bubble up as an `INVALID_ARGUMENT` MCP error result (the existing `convert.ErrorResult` path).

### 5. Error mapping

**Decision:** Reuse the existing `convert.ErrorResult(err)` pattern. gRPC error codes map to MCP error results as follows (these are what the existing 49 tools already do — no new mapping):

| gRPC Code             | MCP error result text                                                                |
| --------------------- | ------------------------------------------------------------------------------------ |
| `NOT_FOUND`           | "not found" (generic; no leakage of which UUID was missing)                          |
| `PERMISSION_DENIED`   | "permission denied" (covers both auth-failure and cross-org access)                  |
| `INVALID_ARGUMENT`    | "invalid argument" (covers malformed UUIDs, unknown enums, missing required fields)  |
| `UNAUTHENTICATED`     | "unauthenticated" (token missing/expired)                                            |
| `INTERNAL`            | "internal error"                                                                     |
| network error         | "connection error: <error>"                                                          |

The detailed gRPC error message is logged server-side (pidgr-mcp side) at WARN level but NOT surfaced in the MCP error result body. This matches the existing tools' "Backend error response" requirement.

**Note for `upsert_reachability`:** A `region_constraint` that fails server-side residency policy returns `INVALID_ARGUMENT` from the upstream service. The MCP error result simply says "invalid argument" — the agent retries with a different region or asks the user.

### 6. Pagination

**Decision:** None of the six new tools paginate. `list_reachabilities_for_user` returns up to N entries (one per `ChannelName` configured for the user — bounded by the enum cardinality, ~8 in Wave 1). The cost-cap and archetype-breakdown RPCs return a single object. No `page_size` / `page_token` arguments are added.

**Why:** The proto definitions don't paginate these calls. The Wave 1 admin slice doesn't need it. Wave 2 may introduce a paginated `ListReachabilitiesForOrg` — but that's a separate RPC and a separate MCP tool.

### 7. `get_campaign_archetype_breakdown` lives in `internal/tools/campaigns.go`, not `internal/tools/insights.go`

**Decision:** The RPC is on `CampaignService`, so its tool registration goes in `campaigns.go` next to `get_campaign`. The five integrations tools live in a new `internal/tools/integrations.go`.

**Why:** The file boundary follows proto-service boundaries (per decision 6 of the original `2026-02-27-mcp-server` change in pidgr-admin's archive). `CampaignService.GetCampaignArchetypeBreakdown` is a campaign RPC even though it surfaces archetype-shift data; the file owner is the service, not the topic.

### 8. Capability lives in pidgr-mcp's own openspec, not pidgr-admin's

**Decision:** This change creates pidgr-mcp's first OpenSpec capability (`mcp-tools`). It does NOT touch pidgr-admin's `openspec/specs/mcp-tools/spec.md`. Going forward, pidgr-mcp owns its own tool spec.

**Why:** pidgr-mcp's `CLAUDE.md` historically pointed at pidgr-admin's OpenSpec because there was no openspec setup in pidgr-mcp. Wave 1 is a natural moment to give pidgr-mcp its own spec home — the new tools cross a service boundary (pidgr-api → pidgr-integrations) that pidgr-admin's spec set has no narrative for.

**Migration of the existing 49 tools:** Out of scope for this change. Follow-up housekeeping change moves the requirements verbatim from `pidgr-admin/openspec/specs/mcp-tools/spec.md` into `pidgr-mcp/openspec/specs/mcp-tools/spec.md`.

**Open decision for user:** Confirm the capability lives in pidgr-mcp's own openspec. Alternative: keep using pidgr-admin's spec (then this change file moves there). The proposal goes with the in-pidgr-mcp option.

## Risks / Trade-offs

- **Empty `specs/mcp-tools` baseline:** Because pidgr-mcp's openspec is brand-new, the `## ADDED Requirements` in this change's delta spec are added to a capability with no pre-existing requirements. `openspec validate --strict` requires deltas to compose against a baseline. We seed the baseline `openspec/specs/mcp-tools/spec.md` with a one-line `## Purpose` and an empty `## Requirements` section (no requirements yet) so the validator can read both the baseline and the delta. The follow-up housekeeping change will populate the baseline with the existing 49-tool surface.
- **Implementation depends on Wave 1 task 5.2:** This is a spec-only PR. The implementation PR cannot land until pidgr-integrations is up. The wave1 checklist tracks the dependency.
- **Capability fragmentation across repos:** Both pidgr-admin and pidgr-mcp now have a `mcp-tools` capability. They describe the same logical surface from different angles (admin: "these tools exist"; mcp: "these tools are registered here"). After the housekeeping migration, only pidgr-mcp's version remains.

## Migration Plan

1. Land this spec-scaffold PR.
2. Wait for Wave 1 task 5.2 (pidgr-integrations bootstrap implementation).
3. Open the implementation PR adding `internal/tools/integrations.go` + the `Integrations` client + `register.go` wiring + the `get_campaign_archetype_breakdown` tool in `campaigns.go` + `internal/tools/schema_test.go` updates + README updates.
4. Mark this OpenSpec change as `archive`d once the implementation PR ships.
5. Open the housekeeping follow-up change that migrates the existing 49 tools' spec into pidgr-mcp's own `mcp-tools` capability and removes them from pidgr-admin's.

## Open Questions

1. `PIDGR_INTEGRATIONS_URL` vs `PIDGR_API_URL` fallback (decision 2 above) — defer to implementation PR, or pin in this spec? Currently deferred.
2. Confirm the capability lives in pidgr-mcp's own openspec (decision 8 above), not pidgr-admin's.
3. Should `upsert_reachability` enforce `permission_denied` on missing `INTEGRATIONS_WRITE` scope client-side, or trust the upstream service to reject? Currently the spec only requires upstream-side enforcement (matches the rest of the 49 tools).
