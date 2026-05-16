## ADDED Requirements

### Requirement: Reachability tools

The system SHALL expose three MCP tools that proxy the reachability-registry RPCs on `pidgr.v1.IntegrationsService`. All three tools SHALL forward the caller's Cognito JWT (HTTP transport) or API key (stdio transport) as the upstream Bearer token. Cross-org access SHALL return `permission_denied` from the upstream service, surfaced to the MCP client as an MCP error result.

#### Scenario: list_reachabilities_for_user happy path

- **WHEN** `list_reachabilities_for_user` is called with `org_id` and `user_id`
- **THEN** the system calls `pidgr.v1.IntegrationsService.ListReachabilityForUser` and returns the `reachabilities` list as proto-JSON
- **AND** the response SHALL NOT contain plaintext identifiers or envelope ciphertext (the proto type omits both fields by construction)

#### Scenario: list_reachabilities_for_user permission denied

- **WHEN** `list_reachabilities_for_user` is called with `org_id` that the caller does not belong to (the upstream service returns `permission_denied`)
- **THEN** the tool returns a `CallToolResult` with `isError: true` and a generic "permission denied" message
- **AND** the upstream error detail SHALL NOT be exposed to the MCP client

#### Scenario: upsert_reachability happy path

- **WHEN** `upsert_reachability` is called with `org_id`, `user_id`, `channel` (canonical `ChannelName` enum-name string, e.g. `"CHANNEL_NAME_EMAIL"`), `identifier_plaintext`, and optional `region_constraint`
- **THEN** the system calls `pidgr.v1.IntegrationsService.UpsertReachability` and returns the upserted `reachability` metadata as proto-JSON
- **AND** the response SHALL NOT include the plaintext identifier or envelope ciphertext

#### Scenario: upsert_reachability flags identifier_plaintext as sensitive

- **WHEN** an MCP client lists tools and inspects the `upsert_reachability` description
- **THEN** the description SHALL state that `identifier_plaintext` is sensitive and is encrypted server-side
- **AND** pidgr-mcp SHALL NOT log the value of `identifier_plaintext` (no debug-level dump of RPC request bodies)

#### Scenario: upsert_reachability unknown channel

- **WHEN** `upsert_reachability` is called with `channel` set to a string that does not match any `ChannelName` enum name (e.g. `"CHANNEL_NAME_FAX"`)
- **THEN** the tool returns a `CallToolResult` with `isError: true` and a generic "invalid argument" message

#### Scenario: remove_reachability happy path

- **WHEN** `remove_reachability` is called with `org_id`, `user_id`, and `channel`
- **THEN** the system calls `pidgr.v1.IntegrationsService.RemoveReachability` and returns the response with `removed: true` when a row existed

#### Scenario: remove_reachability idempotent (no row existed)

- **WHEN** `remove_reachability` is called for a `(org, user, channel)` tuple that has no registry row
- **THEN** the upstream service returns `removed: false` (NOT a `NOT_FOUND` error)
- **AND** the MCP tool returns a successful `CallToolResult` carrying that proto-JSON response

### Requirement: Cost-cap policy tools

The system SHALL expose two MCP tools that proxy the per-`(org, channel)` cost-cap RPCs on `pidgr.v1.IntegrationsService`. Both tools SHALL forward the caller's Bearer token. `set_cost_cap_policy` requires admin scope on the upstream service.

#### Scenario: get_cost_cap_policy happy path

- **WHEN** `get_cost_cap_policy` is called with `org_id` and `channel`
- **THEN** the system calls `pidgr.v1.IntegrationsService.GetCostCapPolicy` and returns the response (`cap_micros`, `used_micros`, `period_yyyymm`) as proto-JSON

#### Scenario: get_cost_cap_policy returns channel-default when no row exists

- **WHEN** `get_cost_cap_policy` is called for an `(org, channel)` with no per-period row
- **THEN** the upstream service returns the channel-default `cap_micros` (server config) and `used_micros: 0` (NOT a `NOT_FOUND` error)
- **AND** the MCP tool returns a successful `CallToolResult` carrying that proto-JSON response

#### Scenario: set_cost_cap_policy happy path

- **WHEN** `set_cost_cap_policy` is called with `org_id`, `channel`, and `cap_micros`
- **THEN** the system calls `pidgr.v1.IntegrationsService.SetCostCapPolicy` and returns the updated policy as proto-JSON

#### Scenario: set_cost_cap_policy permission denied

- **WHEN** `set_cost_cap_policy` is called by a caller without admin scope on the upstream service (upstream returns `permission_denied`)
- **THEN** the tool returns a `CallToolResult` with `isError: true` and a generic "permission denied" message

### Requirement: Campaign archetype breakdown tool

The system SHALL expose one MCP tool that proxies `pidgr.v1.CampaignService.GetCampaignArchetypeBreakdown`. The response surface SHALL include the email-engagement fields added by the `add-email-open-rate-archetype-enrichment` change on pidgr-api (`email_delivered_count`, `email_open_rate_real`, `email_open_rate_raw` per `ArchetypeShareShift`) — passed through verbatim via `protojson.Marshal` on the generated proto type.

#### Scenario: get_campaign_archetype_breakdown happy path

- **WHEN** `get_campaign_archetype_breakdown` is called with `campaign_id`
- **THEN** the system calls `pidgr.v1.CampaignService.GetCampaignArchetypeBreakdown` and returns the response (`shifts`, `before_snapshot_at`, `after_snapshot_at`, `insufficient_history`) as proto-JSON

#### Scenario: get_campaign_archetype_breakdown surfaces email-engagement fields

- **WHEN** `get_campaign_archetype_breakdown` is called for a campaign whose backend response includes non-zero `email_delivered_count` on at least one `ArchetypeShareShift`
- **THEN** the proto-JSON response carried by the `CallToolResult` SHALL include `email_delivered_count`, `email_open_rate_real`, and `email_open_rate_raw` for each shift row that has them populated (subject to `protojson.Marshal` `EmitUnpopulated: false` — zero-value fields are omitted, consistent with the rest of the tool surface)

#### Scenario: get_campaign_archetype_breakdown insufficient history

- **WHEN** `get_campaign_archetype_breakdown` is called for a campaign whose group has fewer than two clustering snapshots
- **THEN** the upstream service returns `shifts: []` with `insufficient_history: true` (NOT a `NOT_FOUND` error)
- **AND** the MCP tool returns a successful `CallToolResult` carrying that proto-JSON response

#### Scenario: get_campaign_archetype_breakdown not found

- **WHEN** `get_campaign_archetype_breakdown` is called with a `campaign_id` that does not exist in the caller's organization
- **THEN** the upstream service returns `NOT_FOUND`
- **AND** the MCP tool returns a `CallToolResult` with `isError: true` and a generic "not found" message

### Requirement: IntegrationsService client wiring

The system SHALL construct a `pidgrv1connect.IntegrationsServiceClient` on `transport.Clients` for both stdio (static-token) and HTTP (dynamic-token) modes. The client SHALL use `connect.WithGRPC()` and the same Bearer-token interceptor pattern as the other service clients on `transport.Clients`.

#### Scenario: Integrations client present in static-token mode

- **WHEN** `transport.NewStaticTokenClients(baseURL, apiKey)` is called
- **THEN** the returned `*Clients` value has a non-nil `Integrations` field
- **AND** invoking any `IntegrationsService` method on it injects `Authorization: Bearer <apiKey>` on the outbound request

#### Scenario: Integrations client present in dynamic-token mode

- **WHEN** `transport.NewDynamicTokenClients(baseURL)` is called from HTTP transport with an authenticated `auth.TokenInfo` in the request context carrying `raw_token`
- **THEN** the `Integrations` client injects `Authorization: Bearer <raw_token>` on the outbound RPC
