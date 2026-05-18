---
page_title: "anthropic_memory_store Data Source - anthropic"
subcategory: ""
description: |-
  Reads an existing Anthropic memory store by ID.
---

# anthropic_memory_store (Data Source)

Reads an existing Anthropic memory store by ID. Requires `admin_api_key` in the provider block.

## Example Usage

```hcl
data "anthropic_memory_store" "shared" {
  provider = anthropic.admin
  id       = "mst_01abc..."
}
```

## Argument Reference

- `id` - (Required) Memory store ID.

## Attributes Reference

- `type` - Memory store type (e.g. `knowledge_base`).
- `name` - Memory store name.
- `description` - Memory store description.
- `metadata` - Map of string key-value pairs.
- `created_at` - ISO 8601 creation timestamp.
- `updated_at` - ISO 8601 last-updated timestamp.
- `archived_at` - ISO 8601 archival timestamp, or null if active.
