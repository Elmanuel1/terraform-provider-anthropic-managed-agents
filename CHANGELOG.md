# Changelog

All notable changes to this provider are documented here.

---

## [0.4.0] — Unreleased

### Added

- **`anthropic-wif_vault`** — new resource for managing vaults. Vaults are workspace-scoped collections of credentials for end-users. Supports `display_name`, `metadata`, and `force_delete` (archive by default, hard-delete when true). Import by vault ID.
- **`anthropic-wif_vault_credential`** — new resource for managing credentials nested under a vault. Supports both `static_bearer` and `mcp_oauth` auth types via a JSON `auth` field. Write-only secret fields (`token`, `access_token`, `refresh_token`, `client_secret`) are stored in state but never returned by the API — they are preserved across reads automatically. `vault_id` is immutable after creation. Import by `vault_id/credential_id`.
- **`anthropic-wif_memory_store`** — new resource for managing memory stores. Supports `name`, `description`, `metadata`, and `force_delete`. Import by memory store ID.

### Notes

- All three new resources are workspace-scoped and require a `workspace_id` field (Required, forces replacement). The workspace ID is used to mint a WIF bearer token — the same auth mechanism as agents and environments.
- All three support soft-delete (archive) by default. Set `force_delete = true` to permanently delete a resource.

---

## [0.3.7] — 2026-05-17

### Fixed

- **Agent spurious update plans** — `ModifyPlan` now only marks `version` and `updated_at` as unknown when a user-controlled field actually changed. Previously, any plan on an existing agent would show a diff on these fields even when nothing changed.
- **Agent `version` on update** — the update request now reads `version` from prior state (not the plan), preventing "value is required" API errors caused by the plan holding an unknown version value.
- **Agent `tools` plan drift** — `marshalJSONList` strips API-injected `configs` and `default_config` keys from tool objects so the stored state matches what the user specified.
- **Environment `packages` plan drift** — `normalizePackages` strips the API-injected `type` key and empty package manager arrays from the packages response, preventing a perpetual diff after the first apply.
- **Environment `packages` JSON shape** — changed `Packages` field in `EnvironmentResponse` from `map[string][]string` to `json.RawMessage` to handle the API returning a richer object than expected.

---

## [0.3.6] — 2026-05-16

### Fixed

- **WIF JTI reuse on parallel creates** — added a `sync.Map` token cache to `WIFConfig` keyed by workspace ID. Parallel Terraform resource creates now share a single minted token instead of each minting their own, preventing `jti_reused` 401 errors from the token endpoint.

---

## [0.3.5] — 2026-05-16

### Fixed

- **Agent `model_speed` default** — `fast` is not supported for `claude-sonnet-4-6`; default changed to `standard` and validation tightened.
- **Environment `packages` unmarshal** — fixed panic when API returned a packages object with a `type` field that couldn't be decoded as `map[string][]string`.

---

## [0.3.4] — 2026-05-15

### Fixed

- **Environment update and archive** — update was missing the `requireWIF` guard; archive endpoint now correctly handles 204 responses.
- **Environment `networking_type` default** — restored `"unrestricted"` as the default to avoid sending an empty string to the API on plan.

---

## [0.3.3] — 2026-05-15

### Added

- **Environment `force_delete`** — boolean field on environment resource. When `false` (default) the environment is archived on destroy; when `true` it is permanently deleted.
- **Agent `mcp_servers`, `skills`, `multiagent`** — new optional JSON string fields on the agent resource mirroring the API's MCP server list, skills list, and multi-agent coordinator config.

### Fixed

- **Agent JSON array fields** — `UseStateForUnknown` added to `tools`, `mcp_servers`, `skills`, and `multiagent` to prevent spurious unknown diffs on no-op plans.
- **Agent array field clearing** — empty JSON arrays (`[]`) are now correctly distinguished from null/absent values.

---

## [0.3.2] — 2026-05-14

### Fixed

- **TFC workload identity token fallback** — provider now falls back to `TFC_WORKLOAD_IDENTITY_TOKEN` when `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` is not set, supporting older TFC agent versions.

---

## [0.3.1] — 2026-05-14

### Fixed

- **Environment in-place updates** — environment resource now updates in place instead of replacing. Description field added. Archive on destroy instead of hard-delete.

---

## [0.3.0] — 2026-05-13

### Added

- **`anthropic-wif_workspace`** — resource for managing Anthropic workspaces. Supports name updates in place. Import by workspace name (resolved to ID server-side).
- **`anthropic-wif_agent`** — resource for managing agents. Supports `name`, `model`, `model_speed`, `system`, `description`, `tools`, `metadata`. Optimistic locking via `version` field. Import by `workspace_id/agent_id`.
- **`anthropic-wif_environment`** — resource for managing execution environments. Supports `networking_type` (`unrestricted` or `limited`), `allowed_hosts`, `allow_mcp_servers`, `allow_package_managers`, `packages`. Import by `workspace_id/environment_id`.
- **WIF (Workload Identity Federation)** — provider exchanges a TFC-injected OIDC JWT for a workspace-scoped bearer token via the Anthropic federation endpoint. Configured via `ANTHROPIC_FEDERATION_RULE_ID`, `ANTHROPIC_ORGANIZATION_ID`, `ANTHROPIC_SERVICE_ACCOUNT_ID`.

---

## [0.2.x] — 2026-05-12

Initial internal releases covering package structure, module naming, and registry configuration.
