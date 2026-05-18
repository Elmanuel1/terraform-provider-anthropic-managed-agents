---
page_title: "anthropic_vault Data Source - anthropic"
subcategory: ""
description: |-
  Reads an existing Anthropic vault by ID.
---

# anthropic_vault (Data Source)

Reads an existing Anthropic vault by ID.

## Example Usage

```hcl
data "anthropic_vault" "mcp_creds" {
  provider     = anthropic.wif
  id           = "vlt_01abc..."
  workspace_id = var.workspace_id
}
```

## Argument Reference

- `id` - (Required) Vault ID.
- `workspace_id` - (Optional) Workspace ID. Required when using WIF authentication.

## Attributes Reference

- `display_name` - Vault display name.
- `metadata` - Map of string key-value pairs.
- `created_at` - ISO 8601 creation timestamp.
- `updated_at` - ISO 8601 last-updated timestamp.
- `archived_at` - ISO 8601 archival timestamp, or null if active.
