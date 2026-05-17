# Changelog

All notable changes to this provider are documented here.

---

## [0.4.4] (2026-05-17)

### Fixed

- **`anthropic-wif_memory_store`**: provider now sends `anthropic-beta: managed-agents-2026-04-01` when authenticating with the admin API key. The previous `admin-api-2025-05-21` beta was rejected by the `/v1/memory_stores` endpoint with HTTP 401.
- **`AdminAPIKey`**: added optional `Beta` field to allow per-caller beta header override. Defaults to `admin-api-2025-05-21` for existing callers.

---

## [0.4.3] (2026-05-17)

### Fixed

- **`anthropic-wif_memory_store`**: switched auth from WIF bearer to admin API key. The `/v1/memory_stores` endpoint rejected WIF bearer tokens with HTTP 401; admin API key is the correct credential.
- **`anthropic-wif_memory_store`**: removed `workspace_id` attribute (not required by the API endpoint).
- **`anthropic-wif_vault_credential`**: strip trailing slash from `mcp_server_url` returned by the API. The API echoes URLs with a trailing slash causing Terraform to report "provider produced inconsistent result" on create.

---

## [0.4.2] (2026-05-17)

### Fixed

- **`anthropic-wif_vault_credential`**: write-only fields (`token`, `access_token`, `refresh_token`, `client_secret`) are now correctly read from `req.Config` in both `Create` and `Update`. These fields are absent from the plan's new state per terraform-plugin-framework semantics, so reading from `req.Plan` only caused them to be empty in the API request body (HTTP 400 `auth.token: Field required`).

---

## [0.4.1] (2026-05-17)

### Fixed

- Retracted. Superseded by v0.4.2.

---

## [0.4.0] (2026-05-17)

### Added

- **`anthropic-wif_vault`**: workspace-scoped vault for storing MCP server credentials. Supports `display_name`, `metadata`, and `force_delete`. Import by `workspace_id/vault_id`.
- **`anthropic-wif_vault_credential`**: credential nested under a vault. Supports `static_bearer` and `mcp_oauth` auth types. Write-only secret fields (`token`, `access_token`, `refresh_token`, `client_secret`) are never stored in state. Import by `workspace_id/vault_id/credential_id`.
- **`anthropic-wif_memory_store`**: memory store for agent persistence. Supports `name`, `description`, `metadata`, and `force_delete`. Import by `workspace_id/memory_store_id`.
- All new resources authenticate via WIF bearer token and support soft-delete (archive) by default.

---

## [0.3.7] (2026-05-17)

### Fixed

- **Agent spurious update plans**: `ModifyPlan` now only marks `version` and `updated_at` as unknown when a user-controlled field actually changed. Previously, any plan on an existing agent would show a diff on these fields even when nothing changed.
- **Agent `version` on update**: the update request now reads `version` from prior state (not the plan), preventing "value is required" API errors caused by the plan holding an unknown version value.
- **Agent `tools` plan drift**: `marshalJSONList` strips API-injected `configs` and `default_config` keys from tool objects so the stored state matches what the user specified.
- **Environment `packages` plan drift**: `normalizePackages` strips the API-injected `type` key and empty package manager arrays from the packages response, preventing a perpetual diff after the first apply.
- **Environment `packages` JSON shape**: changed `Packages` field in `EnvironmentResponse` from `map[string][]string` to `json.RawMessage` to handle the API returning a richer object than expected.

---

## [0.3.6] (2026-05-16)

### Fixed

- **WIF JTI reuse on parallel creates**: added a `sync.Map` token cache to `WIFConfig` keyed by workspace ID. Parallel Terraform resource creates now share a single minted token instead of each minting their own, preventing `jti_reused` 401 errors from the token endpoint.

---

## [0.3.5] (2026-05-16)

### Fixed

- **Agent `model_speed` default**: `fast` is not supported for `claude-sonnet-4-6`; default changed to `standard` and validation tightened.
- **Environment `packages` unmarshal**: fixed panic when API returned a packages object with a `type` field that couldn't be decoded as `map[string][]string`.

---

## [0.3.4] (2026-05-15)

### Fixed

- **Environment update and archive**: update was missing the `requireWIF` guard; archive endpoint now correctly handles 204 responses.
- **Environment `networking_type` default**: restored `"unrestricted"` as the default to avoid sending an empty string to the API on plan.

---

## [0.3.3] (2026-05-15)

### Added

- **Environment `force_delete`**: boolean field on environment resource. When `false` (default) the environment is archived on destroy; when `true` it is permanently deleted.
- **Agent `mcp_servers`, `skills`, `multiagent`**: new optional JSON string fields on the agent resource mirroring the API's MCP server list, skills list, and multi-agent coordinator config.

### Fixed

- **Agent JSON array fields**: `UseStateForUnknown` added to `tools`, `mcp_servers`, `skills`, and `multiagent` to prevent spurious unknown diffs on no-op plans.
- **Agent array field clearing**: empty JSON arrays (`[]`) are now correctly distinguished from null/absent values.

---

## [0.3.2] (2026-05-14)

### Fixed

- **TFC workload identity token fallback**: provider now falls back to `TFC_WORKLOAD_IDENTITY_TOKEN` when `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` is not set, supporting older TFC agent versions.

---

## [0.3.1] (2026-05-14)

### Fixed

- **Environment in-place updates**: environment resource now updates in place instead of replacing. Description field added. Archive on destroy instead of hard-delete.

---

## [0.3.0] (2026-05-13)

### Added

- **`anthropic-wif_workspace`**: resource for managing Anthropic workspaces. Supports name updates in place. Import by workspace name (resolved to ID server-side).
- **`anthropic-wif_agent`**: resource for managing agents. Supports `name`, `model`, `model_speed`, `system`, `description`, `tools`, `metadata`. Optimistic locking via `version` field. Import by `workspace_id/agent_id`.
- **`anthropic-wif_environment`**: resource for managing execution environments. Supports `networking_type` (`unrestricted` or `limited`), `allowed_hosts`, `allow_mcp_servers`, `allow_package_managers`, `packages`. Import by `workspace_id/environment_id`.
- **WIF (Workload Identity Federation)**: provider exchanges a TFC-injected OIDC JWT for a workspace-scoped bearer token via the Anthropic federation endpoint. Configured via `ANTHROPIC_FEDERATION_RULE_ID`, `ANTHROPIC_ORGANIZATION_ID`, `ANTHROPIC_SERVICE_ACCOUNT_ID`.

---

## [0.2.x] (2026-05-12)

Initial internal releases covering package structure, module naming, and registry configuration.
