## 1. Transport wiring

- [ ] 1.1 Add `Integrations pidgrv1connect.IntegrationsServiceClient` field to `transport.Clients` in `internal/transport/clients.go`.
- [ ] 1.2 Construct the Integrations client in `newClients` using `pidgrv1connect.NewIntegrationsServiceClient(httpClient, baseURL, grpc, opts)`.
- [ ] 1.3 (Decide before implementing) Resolve the IntegrationsService base URL — either `PIDGR_INTEGRATIONS_URL` env var with fallback to `PIDGR_API_URL`, or hard-require `PIDGR_INTEGRATIONS_URL`. Land the chosen rule in `cmd/pidgr-mcp/main.go` and wire it into both `NewStaticTokenClients` and `NewDynamicTokenClients` callsites.
- [ ] 1.4 Update `internal/transport/clients_test.go` to cover the new Integrations client: a) it exists in both static and dynamic mode, b) the Bearer token is injected on outbound RPCs.

## 2. Integrations tools (`internal/tools/integrations.go`)

- [ ] 2.1 Create `internal/tools/integrations.go` with the copyright header and package declaration matching the rest of `internal/tools/`.
- [ ] 2.2 Define input structs:
  - `ListReachabilitiesForUserInput { OrgID, UserID string }`
  - `UpsertReachabilityInput { OrgID, UserID, Channel, IdentifierPlaintext string; RegionConstraint *string }`
  - `RemoveReachabilityInput { OrgID, UserID, Channel string }`
  - `GetCostCapPolicyInput { OrgID, Channel string }`
  - `SetCostCapPolicyInput { OrgID, Channel string; CapMicros int64 }`
- [ ] 2.3 Implement `registerIntegrationsTools(s *mcp.Server, c *transport.Clients)`:
  - Register all five tools via `mcp.AddTool(s, &mcp.Tool{Name, Description}, handler)`.
  - Each handler: call the corresponding `c.Integrations.*` method via `connect.NewRequest(...)`, fold errors via `convert.ErrorResult`, success via `convert.ProtoResult`.
- [ ] 2.4 Channel-arg parsing: convert the JSON string `channel` to `pidgrv1.ChannelName` via the generated `pidgrv1.ChannelName_value` map (or equivalent). Return an `INVALID_ARGUMENT` error result on unknown values.
- [ ] 2.5 `upsert_reachability` description SHALL state that `identifier_plaintext` is sensitive and is encrypted server-side.
- [ ] 2.6 Wire `registerIntegrationsTools(s, c)` into `RegisterAll` in `internal/tools/register.go`.

## 3. Campaign archetype breakdown tool (`internal/tools/campaigns.go`)

- [ ] 3.1 Add `GetCampaignArchetypeBreakdownInput { CampaignID string }` to `internal/tools/campaigns.go`.
- [ ] 3.2 Register the `get_campaign_archetype_breakdown` tool inside the existing `registerCampaignTools` function with a description mentioning that the response includes the Wave 1 email-engagement fields (`email_delivered_count`, `email_open_rate_real`, `email_open_rate_raw` per `ArchetypeShareShift`).
- [ ] 3.3 Handler: call `c.Campaigns.GetCampaignArchetypeBreakdown(ctx, connect.NewRequest(&pidgrv1.GetCampaignArchetypeBreakdownRequest{CampaignId: input.CampaignID}))`, fold errors + success via the same `convert.ErrorResult` / `convert.ProtoResult` path as the other campaign tools.

## 4. Tests

- [ ] 4.1 Update `internal/tools/schema_test.go` (or its successor) to include the six new tools in the expected-tools list.
- [ ] 4.2 Update `internal/tools/register_test.go` to assert that `RegisterAll` registers the six new tools (and only them, in addition to the existing 49).
- [ ] 4.3 Add unit tests in `internal/tools/integrations_test.go` covering: a) happy path for each tool (mock IntegrationsService client returns success); b) `permission_denied` error path for `list_reachabilities_for_user` and `set_cost_cap_policy`; c) `INVALID_ARGUMENT` for unknown `channel` string on `upsert_reachability`; d) `NOT_FOUND` for `get_campaign_archetype_breakdown`.
- [ ] 4.4 Add a regression assertion that pidgr-mcp does NOT log the `identifier_plaintext` value of `upsert_reachability` (introspect the logger output buffer in a test that calls the tool with a sentinel value).

## 5. Documentation

- [ ] 5.1 Update `README.md` tool table: bump count from 49 to 55, add the six new tool names + one-line descriptions.
- [ ] 5.2 Update `README.md` env-var table to document `PIDGR_INTEGRATIONS_URL` (per the decision landed in 1.3).
- [ ] 5.3 Update `CLAUDE.md` to note that pidgr-mcp now owns its OpenSpec setup (replace "Changes for this repo are tracked in pidgr-admin's OpenSpec" with the path to the local `openspec/` setup).

## 6. Validation + release

- [ ] 6.1 Run `openspec validate wave1-mcp-channels-and-reachability-tools --strict` and confirm it passes.
- [ ] 6.2 Run `go vet ./...`, `golangci-lint run`, `go test ./... -cover` — all green.
- [ ] 6.3 Open implementation PR; reference this spec scaffold PR in the body.
- [ ] 6.4 After implementation PR merges, run `openspec archive wave1-mcp-channels-and-reachability-tools` to move the change to `openspec/changes/archive/` and sync the spec into `openspec/specs/mcp-tools/spec.md`.

## 7. Follow-up (out of scope for this change)

- [ ] 7.1 Open a housekeeping change `migrate-existing-49-tools-spec-into-pidgr-mcp` that copies the existing 49-tool requirements from pidgr-admin's `openspec/specs/mcp-tools/spec.md` into pidgr-mcp's own capability and removes them from pidgr-admin's spec set.
- [ ] 7.2 Once Wave 2 introduces additional channel-management RPCs (e.g. `GetRegionPolicy` / `SetRegionPolicy` admin tooling), open a separate change adding the corresponding MCP tools.
