---
page_title: "anthropic_vault_credential Data Source - anthropic"
subcategory: ""
description: |-
  Reads an existing Anthropic vault credential by vault ID and credential ID.
---

# anthropic_vault_credential (Data Source)

Reads an existing Anthropic vault credential. Secret fields (`token`, `access_token`, `refresh_token`, `client_secret`) are never returned by the API and will not appear in state.

## Example Usage

```hcl
data "anthropic_vault_credential" "bearer" {
  provider     = anthropic.wif
  vault_id     = var.vault_id
  id           = "vcrd_01abc..."
  workspace_id = var.workspace_id
}
```

## Argument Reference

- `id` - (Required) Credential ID.
- `vault_id` - (Required) ID of the vault this credential belongs to.
- `workspace_id` - (Optional) Workspace ID. Required when using WIF authentication.

## Attributes Reference

- `display_name` - Credential display name.
- `auth_type` - Credential type: `static_bearer` or `mcp_oauth`.
- `mcp_server_url` - MCP server URL.
- `expires_at` - OAuth token expiry timestamp.
- `token_endpoint` - OAuth token endpoint URL.
- `client_id` - OAuth client ID.
- `token_endpoint_auth_type` - OAuth token endpoint auth method.
- `scope` - OAuth scope.
- `resource` - OAuth resource indicator.
- `metadata` - Map of string key-value pairs.
- `created_at` - ISO 8601 creation timestamp.
- `updated_at` - ISO 8601 last-updated timestamp.
- `archived_at` - ISO 8601 archival timestamp, or null if active.
