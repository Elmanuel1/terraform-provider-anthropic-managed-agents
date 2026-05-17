---
page_title: "anthropic-wif_vault_credential Resource"
description: |-
  Manages a credential stored in an Anthropic vault.
---

# anthropic-wif_vault_credential

Manages a credential inside an Anthropic vault. Credentials provide MCP server authentication for agents. Both static bearer tokens and OAuth flows are supported.

Secret fields (`token`, `access_token`, `refresh_token`, `client_secret`) are **write-only**: they are sent to the API on create/update but never stored in Terraform state and never returned by reads.

Authenticates via WIF bearer token scoped to the `workspace_id`.

On destroy the credential is **archived** by default. Set `force_delete = true` to permanently delete it.

## Example Usage

### Static bearer token

```terraform
resource "anthropic-wif_vault_credential" "example" {
  workspace_id   = anthropic-wif_workspace.example.id
  vault_id       = anthropic-wif_vault.example.id
  display_name   = "my-mcp-server-token"
  auth_type      = "static_bearer"
  mcp_server_url = "https://mcp.example.com"
  token          = var.mcp_token  # write-only, never stored in state
}
```

### OAuth with refresh

```terraform
resource "anthropic-wif_vault_credential" "example" {
  workspace_id   = anthropic-wif_workspace.example.id
  vault_id       = anthropic-wif_vault.example.id
  display_name   = "my-oauth-credential"
  auth_type      = "mcp_oauth"
  mcp_server_url = "https://mcp.example.com"

  access_token  = var.access_token   # write-only
  refresh_token = var.refresh_token  # write-only
  expires_at    = "2026-12-31T00:00:00Z"

  token_endpoint          = "https://auth.example.com/token"
  client_id               = "my-client-id"
  token_endpoint_auth_type = "client_secret_post"
  client_secret           = var.client_secret  # write-only
  scope                   = "read write"
}
```

## Import

Import by `workspace_id/vault_id/credential_id`:

```shell
terraform import anthropic-wif_vault_credential.example wrks_xxx/vlt_yyy/vcrd_zzz
```

~> **Note:** Secret fields (`token`, `access_token`, `refresh_token`, `client_secret`) cannot be recovered from state after import. You must re-apply with the values set to restore them in the API.

## Argument Reference

| Argument | Type | Required | Description |
|---|---|---|---|
| `workspace_id` | string | Yes | Workspace ID. Changing this forces a new resource. |
| `vault_id` | string | Yes | Vault ID. Changing this forces a new resource. |
| `auth_type` | string | Yes | `static_bearer` or `mcp_oauth`. Changing this forces a new resource. |
| `mcp_server_url` | string | Yes | MCP server URL. Changing this forces a new resource. |
| `display_name` | string | No | Human-readable credential name. |
| `token` | string | No | **Write-only.** Static bearer token. Required when `auth_type = "static_bearer"`. |
| `access_token` | string | No | **Write-only.** OAuth access token. Used when `auth_type = "mcp_oauth"`. |
| `refresh_token` | string | No | **Write-only.** OAuth refresh token. Used when `auth_type = "mcp_oauth"`. |
| `expires_at` | string | No | OAuth token expiry timestamp (ISO 8601). |
| `token_endpoint` | string | No | OAuth token endpoint URL. Changing this forces a new resource. |
| `client_id` | string | No | OAuth client ID. Changing this forces a new resource. |
| `token_endpoint_auth_type` | string | No | OAuth token endpoint auth method: `none`, `client_secret_basic`, or `client_secret_post`. Changing this forces a new resource. |
| `client_secret` | string | No | **Write-only.** OAuth client secret. |
| `scope` | string | No | OAuth scope string. |
| `resource` | string | No | OAuth resource indicator. |
| `metadata` | map(string) | No | Arbitrary string key-value pairs. |
| `force_delete` | bool | No | When `true`, permanently deletes on destroy. Default `false` (archives). |

## Attribute Reference

| Attribute | Type | Description |
|---|---|---|
| `id` | string | Credential ID (`vcrd_...`). |
| `created_at` | string | ISO 8601 creation timestamp. |
| `updated_at` | string | ISO 8601 last-updated timestamp. |
| `archived_at` | string | ISO 8601 archival timestamp, or null if active. |
