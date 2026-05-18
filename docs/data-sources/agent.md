---
page_title: "anthropic_agent Data Source - anthropic"
subcategory: ""
description: |-
  Reads an existing Anthropic agent by ID.
---

# anthropic_agent (Data Source)

Reads an existing Anthropic agent by ID.

## Example Usage

```hcl
data "anthropic_agent" "procurement" {
  provider     = anthropic.wif
  id           = "agt_01abc..."
  workspace_id = var.workspace_id
}

output "agent_model" {
  value = data.anthropic_agent.procurement.model
}
```

## Argument Reference

- `id` - (Required) Agent ID (`agt_...`).
- `workspace_id` - (Optional) Workspace ID. Required when using WIF authentication.

## Attributes Reference

- `name` - Agent name.
- `model` - Model ID (e.g. `claude-sonnet-4-6`).
- `model_speed` - `standard` or `fast`.
- `system` - System prompt.
- `description` - Agent description.
- `tools` - JSON-encoded tools array.
- `mcp_servers` - JSON-encoded MCP servers array.
- `skills` - JSON-encoded skills array.
- `multiagent` - JSON-encoded multiagent config.
- `metadata` - Map of string key-value pairs.
- `version` - Agent version number.
- `created_at` - ISO 8601 creation timestamp.
- `updated_at` - ISO 8601 last-updated timestamp.
- `archived_at` - ISO 8601 archival timestamp, or null if active.
