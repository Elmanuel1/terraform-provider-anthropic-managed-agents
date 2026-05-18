---
page_title: "anthropic: anthropic_wif_vault"
subcategory: ""
description: |-
  Manages an Anthropic vault for storing MCP server credentials.
---

# Resource: anthropic_wif_vault

Manages an Anthropic vault. Vaults are workspace-scoped containers for storing MCP server credentials that agents can use during sessions.

Authenticates via WIF bearer token scoped to the `workspace_id`.

On destroy the vault is archived by default. Set `force_delete = true` to permanently delete it.

## Example Usage

```terraform
resource "anthropic_wif_vault" "example" {
  workspace_id = anthropic_workspace.example.id
  display_name = "production-vault"

  metadata = {
    env  = "production"
    team = "platform"
  }
}
```

## Argument Reference

This resource supports the following arguments:

* `workspace_id` - (Required, Forces new resource) Workspace ID.
* `display_name` - (Required) Human-readable vault name.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.
* `force_delete` - (Optional) When `true`, permanently deletes on destroy. Default `false` (archives).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Vault ID (`vlt_...`).
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by `workspace_id/vault_id`:

```shell
terraform import anthropic_wif_vault.example wrks_xxx/vlt_yyy
```
