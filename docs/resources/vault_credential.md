---
page_title: "anthropic: anthropic_vault_credential"
subcategory: ""
description: |-
  Manages a credential stored in an Anthropic vault.
---

# Resource: anthropic_vault_credential

Manages a credential inside an Anthropic vault. Credentials provide MCP server authentication for agents. Both static bearer tokens and OAuth flows are supported.

Secret fields (`token`, `access_token`, `refresh_token`, `client_secret`) are write-only: they are sent to the API on create/update but never stored in Terraform state and never returned by reads.

Authenticates via WIF bearer token scoped to the `workspace_id`.

On destroy the credential is archived by default. Set `force_delete = true` to permanently delete it.

## Example Usage

### Static bearer token

```terraform
resource "anthropic_vault_credential" "example" {
  workspace_id   = anthropic_workspace.example.id
  vault_id       = anthropic_vault.example.id
  display_name   = "my-mcp-server-token"
  auth_type      = "static_bearer"
  mcp_server_url = "https://mcp.example.com"
  token          = var.mcp_token
}
```

### OAuth with refresh

```terraform
resource "anthropic_vault_credential" "example" {
  workspace_id   = anthropic_workspace.example.id
  vault_id       = anthropic_vault.example.id
  display_name   = "my-oauth-credential"
  auth_type      = "mcp_oauth"
  mcp_server_url = "https://mcp.example.com"

  access_token  = var.access_token
  refresh_token = var.refresh_token
  expires_at    = "2026-12-31T00:00:00Z"

  token_endpoint           = "https://auth.example.com/token"
  client_id                = "my-client-id"
  token_endpoint_auth_type = "client_secret_post"
  client_secret            = var.client_secret
  scope                    = "read write"
}
```

## Argument Reference

This resource supports the following arguments:

* `workspace_id` - (Required, Forces new resource) Workspace ID.
* `vault_id` - (Required, Forces new resource) Vault ID.
* `auth_type` - (Required, Forces new resource) `static_bearer` or `mcp_oauth`.
* `mcp_server_url` - (Required, Forces new resource) MCP server URL.
* `display_name` - (Optional) Human-readable credential name.
* `token` - (Optional, Write-only) Static bearer token. Required when `auth_type = "static_bearer"`.
* `access_token` - (Optional, Write-only) OAuth access token. Used when `auth_type = "mcp_oauth"`.
* `refresh_token` - (Optional, Write-only) OAuth refresh token. Used when `auth_type = "mcp_oauth"`.
* `expires_at` - (Optional) OAuth token expiry timestamp (ISO 8601).
* `token_endpoint` - (Optional, Forces new resource) OAuth token endpoint URL.
* `client_id` - (Optional, Forces new resource) OAuth client ID.
* `token_endpoint_auth_type` - (Optional, Forces new resource) OAuth token endpoint auth method: `none`, `client_secret_basic`, or `client_secret_post`.
* `client_secret` - (Optional, Write-only) OAuth client secret.
* `scope` - (Optional) OAuth scope string.
* `resource` - (Optional) OAuth resource indicator.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.
* `force_delete` - (Optional) When `true`, permanently deletes on destroy. Default `false` (archives).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Credential ID (`vcrd_...`).
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by `workspace_id/vault_id/credential_id`:

```shell
terraform import anthropic_vault_credential.example wrks_xxx/vlt_yyy/vcrd_zzz
```

~> **Note:** Write-only fields (`token`, `access_token`, `refresh_token`, `client_secret`) cannot be recovered from state after import. Re-apply with the values set to restore them in the API.
