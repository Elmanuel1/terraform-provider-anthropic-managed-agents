---
page_title: "anthropic_environment Data Source - anthropic"
subcategory: ""
description: |-
  Reads an existing Anthropic environment by ID.
---

# anthropic_environment (Data Source)

Reads an existing Anthropic environment by ID.

## Example Usage

```hcl
data "anthropic_environment" "sandbox" {
  provider     = anthropic.wif
  id           = "env_01abc..."
  workspace_id = var.workspace_id
}
```

## Argument Reference

- `id` - (Required) Environment ID.
- `workspace_id` - (Optional) Workspace ID. Required when using WIF authentication.

## Attributes Reference

- `name` - Environment name.
- `description` - Environment description.
- `networking_type` - `unrestricted` or `limited`.
- `allowed_hosts` - List of allowed outbound hosts (when `networking_type` is `limited`).
- `allow_mcp_servers` - Whether MCP server network access is allowed.
- `allow_package_managers` - Whether package manager network access is allowed.
- `packages` - JSON-encoded packages map.
- `metadata` - Map of string key-value pairs.
- `created_at` - ISO 8601 creation timestamp.
- `updated_at` - ISO 8601 last-updated timestamp.
- `archived_at` - ISO 8601 archival timestamp, or null if active.
