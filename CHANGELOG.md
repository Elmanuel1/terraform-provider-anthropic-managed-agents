# Changelog

All notable changes to this provider are documented here.

---

## [Unreleased]

### Added

- `anthropic_environment` resource and data source now support a `type` attribute (`"cloud"` or `"self_hosted"`, default `"cloud"`). Changing `type` forces a new resource.
- `anthropic_environment` resource and data source now expose a `scope` attribute (`"organization"` or `"account"`). Only meaningful for `self_hosted` environments; the API ignores it for `cloud` environments.

---

## [0.3.1] (2026-05-22)

### Fixed

- `tools`, `mcp_servers`, `skills`, `multiagent` on `anthropic_agent` and `packages` on `anthropic_environment` now use `jsontypes.Normalized` instead of plain `types.String`. This fixes perpetual plan diffs caused by JSON key-ordering differences between the plan value and the API response.
- The above fields now validate that their value is valid JSON at plan time. Previously any string was accepted silently.

---

## [0.1.0] (2026-05-18)

Initial release of `Elmanuel1/anthropic`.

### Added

- **`anthropic_workspace`**: create and manage Anthropic workspaces. Authenticates with `admin_api_key`. Import by workspace name.
- **`anthropic_memory_store`**: memory store for agent persistence. Authenticates with `admin_api_key`. Supports `name`, `description`, `metadata`, `force_delete`. Import by `memory_store_id`.
- **`anthropic_agent`**: create and manage Anthropic agents. Supports two auth modes: WIF (`workspace_id` + `federation_rule_id` / `organization_id` / `service_account_id`) or workspace API key (`workspace_api_key`). WIF takes precedence when both are configured. Supports `name`, `model`, `model_speed`, `system`, `description`, `tools`, `mcp_servers`, `skills`, `multiagent`, `metadata`. Optimistic locking via `version`. Import by `workspace_id/agent_id` (WIF) or `agent_id` (API key).
- **`anthropic_environment`**: execution environment for agents. Authenticates via WIF. Supports `networking_type`, `allowed_hosts`, `allow_mcp_servers`, `allow_package_managers`, `packages`, `force_delete`. Import by `workspace_id/environment_id`.
- **`anthropic_vault`**: workspace-scoped vault for storing MCP server credentials. Authenticates via WIF. Supports `display_name`, `metadata`, `force_delete`. Import by `workspace_id/vault_id`.
- **`anthropic_vault_credential`**: credential nested under a vault. Authenticates via WIF. Supports `static_bearer` and `mcp_oauth` auth types. Write-only secret fields (`token`, `access_token`, `refresh_token`, `client_secret`) are never stored in state. Import by `workspace_id/vault_id/credential_id`.
- **WIF (Workload Identity Federation)**: provider exchanges a TFC-injected OIDC JWT for a workspace-scoped bearer token. Configured via `federation_rule_id`, `organization_id`, `service_account_id` in the provider block. The JWT is read from `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` (or `TFC_WORKLOAD_IDENTITY_TOKEN` as fallback) — injected automatically by Terraform Cloud.
- **Workspace API key**: `workspace_api_key` provider attribute for `anthropic_agent` authentication without WIF.
- **Plan-time credential validation** on `anthropic_agent`.
- **Token caching**: minted WIF tokens are cached per workspace ID to prevent JTI reuse across parallel resource creates.
