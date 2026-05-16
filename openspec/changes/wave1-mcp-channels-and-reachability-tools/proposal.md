## Why

Ivysaur Wave 1 introduces a new `pidgr.v1.IntegrationsService` (hosted by the sibling `pidgr-integrations` data plane) that exposes the channel-dispatch + reachability + region-policy + cost-cap surface to admin callers over Cognito-JWT. AI agents driving Pidgr from Claude Code, Cursor, or Windsurf have no tools today for any of that surface — they can manage campaigns, templates, members and the rest of the v0.1 RPC set, but they cannot configure a recipient's email/SMS/Slack identifier, inspect a user's per-channel reachability matrix, or read/write the per-org cost-cap policy that gates Wave 1 paid channels.

Wave 1 task 4.3 in `docs/wave1-spec-review-checklist.md` calls for "Wave 1 channel + reachability tools" in pidgr-mcp (~1-2 dev-days, blocker = Wave 1 task 5.2 — pidgr-integrations bootstrap implementation). This change scaffolds the spec only; the implementation PR lands once 5.2 has shipped a callable IntegrationsService at a known endpoint.

Wave 1 task 1.9 (email open-rate archetype enrichment, shipped via `add-email-open-rate-archetype-enrichment` on pidgr-api) added three new fields to `GetCampaignArchetypeBreakdownResponse.shifts` — `email_delivered_count`, `email_open_rate_real`, `email_open_rate_raw`. pidgr-mcp does NOT yet expose `GetCampaignArchetypeBreakdown` at all, so this change also adds it (the new email-engagement fields then pass through naturally via the generated proto types).

## What Changes

- **NEW tool** `list_reachabilities_for_user` → `pidgr.v1.IntegrationsService.ListReachabilityForUser`. Returns per-channel reachability metadata for one `(org, user)` pair (plaintext + ciphertext NEVER returned).
- **NEW tool** `upsert_reachability` → `pidgr.v1.IntegrationsService.UpsertReachability`. Records a recipient identifier (`identifier_plaintext`) for a `(user, channel)` tuple. Plaintext is column-level KMS-encrypted server-side; the server computes the HMAC lookup hash. The tool description SHALL flag this argument as sensitive.
- **NEW tool** `remove_reachability` → `pidgr.v1.IntegrationsService.RemoveReachability`. Idempotent removal of a reachability row. Emits a `REACHABILITY_REMOVE` audit row server-side before deleting.
- **NEW tool** `get_cost_cap_policy` → `pidgr.v1.IntegrationsService.GetCostCapPolicy`. Reads the current calendar-month cap + accumulated spend for `(org, channel)`. Falls back to the channel default cap server-side when no row exists (never NOT_FOUND).
- **NEW tool** `set_cost_cap_policy` → `pidgr.v1.IntegrationsService.SetCostCapPolicy`. Admin-only upsert of the current calendar-month cost cap (micros) for `(org, channel)`.
- **NEW tool** `get_campaign_archetype_breakdown` → `pidgr.v1.CampaignService.GetCampaignArchetypeBreakdown`. Returns the archetype-tendency-shift surface for a campaign. The Wave 1 `email_delivered_count` + `email_open_rate_real` + `email_open_rate_raw` fields on each `ArchetypeShareShift` pass through naturally via the generated proto type — the tool itself just relays the proto response.
- **NEW** `IntegrationsServiceClient` field on `transport.Clients` plus `pidgrv1connect.NewIntegrationsServiceClient(...)` wiring in `newClients` and the corresponding entry in `RegisterAll` (`registerIntegrationsTools(s, c)`).

Two non-Wave-1 IntegrationsService RPCs are intentionally NOT exposed by this change (see design.md):

- `DispatchToChannel` — internal-mTLS-only worker RPC; no Cognito-JWT auth path; not callable from MCP.
- `GetReachability` / `GetRegionPolicy` / `SetRegionPolicy` — deferred. `GetReachability` overlaps with `list_reachabilities_for_user` and isn't on Wave 1's admin slice. Region policy is admin-UI-driven in Wave 1; an MCP tool can be added in a follow-up once 4.2 (Wave 1 admin UI) lands and we have a concrete agent-driven use case.

## Capabilities

### New Capabilities

- `mcp-tools`: First MCP-tools capability for pidgr-mcp's own OpenSpec setup. Mirrors the capability of the same name in pidgr-admin's `2026-02-27-mcp-server` change (which scaffolded the original 49-tool v0.1 surface from the admin side). Going forward this is where pidgr-mcp's tool surface is specified. This change `## ADDED Requirements`-style adds six new tool requirements onto that capability. The pre-existing 49 tools are NOT re-specified by this change; they will be migrated into this repo's capability in a follow-up housekeeping change.

## Impact

- **Code**: New `internal/tools/integrations.go` (registers six new tools; will land in the implementation PR, not this spec scaffold). New entries in `internal/tools/register.go` (`registerIntegrationsTools`) and `internal/tools/campaigns.go` (`get_campaign_archetype_breakdown`). New `Integrations pidgrv1connect.IntegrationsServiceClient` field on `transport.Clients` plus wiring in `newClients` (both static and dynamic transports).
- **Schema**: None. This change consumes proto definitions already shipped via pidgr-proto v0.74.0+ (`integrations_service.proto`, `integrations.proto`, `channel_events.proto::ChannelName`) and the email-open-rate fields already added to `campaign.proto::ArchetypeShareShift` by `add-email-open-rate-archetype-enrichment` in pidgr-api's spec set.
- **Configuration**: New env var `PIDGR_INTEGRATIONS_URL` (Connect-Web base URL for the IntegrationsService endpoint). Falls back to `PIDGR_API_URL` if unset — the implementation PR will pick the resolution rule once 5.2 settles on either co-hosted-on-pidgr-api-ALB or a dedicated `integrations.pidgr.com` hostname. README updated with the new env var and the six new tools.
- **Auth**: All six new tools require a Cognito JWT in the caller's MCP request (HTTP transport) or a Cognito-JWT-equivalent API key in `PIDGR_API_KEY` (stdio transport) with `INTEGRATIONS_READ` / `INTEGRATIONS_WRITE` permissions on the calling org. Cross-org calls return `permission_denied` from the upstream service — pidgr-mcp passes this through as an MCP error result.
- **Out of scope**:
  - The IntegrationsService implementation itself (lives in pidgr-integrations, Wave 1 task 5.2).
  - The `DispatchToChannel` / `GetReachability` / `GetRegionPolicy` / `SetRegionPolicy` RPCs (rationale above).
  - Migrating the existing 49 tools into pidgr-mcp's own OpenSpec capability (housekeeping follow-up).
  - The Wave 1 admin UI for these RPCs (Wave 1 task 4.2, pidgr-admin).
- **Forward-compatibility**: Wave 2 channels (Twilio SMS / Slack / Telegram / WhatsApp / Teams / LINE) are already enumerated in `ChannelName` and reuse the same six MCP tools unchanged — the channel argument is an enum value. The schema definitions in the implementation PR SHALL surface every `ChannelName` enum value to MCP clients via the generated proto type, not via a hand-curated subset.
