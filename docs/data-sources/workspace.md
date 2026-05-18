---
page_title: "anthropic_workspace Data Source - anthropic"
subcategory: ""
description: |-
  Reads an existing Anthropic workspace by ID or name.
---

# anthropic_workspace (Data Source)

Reads an existing Anthropic workspace by ID or name. Requires `admin_api_key` in the provider block.

## Example Usage

```hcl
# Look up by name
data "anthropic_workspace" "by_name" {
  provider = anthropic.admin
  name     = "production"
}

output "workspace_id" {
  value = data.anthropic_workspace.by_name.id
}
```

```hcl
# Look up by ID
data "anthropic_workspace" "by_id" {
  provider = anthropic.admin
  id       = "wrks_01abc..."
}
```

## Argument Reference

One of `id` or `name` must be set.

- `id` - (Optional) Workspace ID (`wrks_...`).
- `name` - (Optional) Workspace name.

## Attributes Reference

- `id` - Workspace ID.
- `name` - Workspace name.
- `created_at` - ISO 8601 creation timestamp.
- `archived_at` - ISO 8601 archival timestamp, or null if active.
