---
page_title: "anthropic_memory_store Resource - anthropic"
subcategory: ""
description: |-
  Manages an Anthropic memory store for agent persistence.
---

# anthropic_memory_store (Resource)

Manages an Anthropic memory store. Memory stores provide persistent storage for agents across sessions, enabling long-term context and knowledge retention.

Authenticates with the Anthropic Admin API key (`ANTHROPIC_ADMIN_API_KEY`).

On destroy the memory store is archived by default. Set `force_delete = true` to permanently delete it.

## Example Usage

```terraform
resource "anthropic_memory_store" "example" {
  name        = "agent-memory"
  description = "Persistent memory for the procurement agent."

  metadata = {
    env  = "production"
    team = "platform"
  }
}
```

## Argument Reference

* `name` - (Required) Memory store name.
* `description` - (Optional) Human-readable description.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.
* `force_delete` - (Optional) When `true`, permanently deletes on destroy. Default `false` (archives).

## Attribute Reference

* `id` - Memory store ID.
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by memory store ID:

```shell
terraform import anthropic_memory_store.example ms_xxx
```
