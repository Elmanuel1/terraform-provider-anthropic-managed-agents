---
page_title: "anthropic_wif_agent Resource - anthropic"
subcategory: ""
description: |-
  Manages an Anthropic agent.
---

# anthropic_wif_agent (Resource)

Manages an Anthropic agent. Agents are workspace-scoped and authenticate via WIF bearer token.

## Example Usage

### Minimal agent

```terraform
resource "anthropic_wif_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "my-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."
}
```

### Agent with tools and MCP servers

```terraform
resource "anthropic_wif_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "procurement-agent"
  model        = "claude-opus-4-7"
  model_speed  = "standard"
  system       = "You are a procurement assistant."
  description  = "Handles purchase order workflows."

  tools = jsonencode([
    { "type" = "agent_toolset_20260401" }
  ])

  mcp_servers = jsonencode([
    {
      name = "erp-server"
      type = "url"
      url  = "https://erp.example.com/mcp"
    }
  ])

  metadata = {
    team = "procurement"
    env  = "production"
  }
}
```

### Multi-agent coordinator

```terraform
resource "anthropic_wif_agent" "coordinator" {
  workspace_id = anthropic_workspace.example.id
  name         = "coordinator"
  model        = "claude-opus-4-7"

  multiagent = jsonencode({
    type   = "coordinator"
    agents = [anthropic_wif_agent.worker.id]
  })
}
```

## Argument Reference

* `workspace_id` - (Required, Forces new resource) Workspace ID.
* `name` - (Required) Agent name.
* `model` - (Required) Model ID, e.g. `claude-opus-4-7` or `claude-sonnet-4-6`.
* `model_speed` - (Optional) Inference speed: `standard` (default) or `fast`.
* `system` - (Optional) System prompt.
* `description` - (Optional) Human-readable description.
* `tools` - (Optional) JSON-encoded tools array. Maximum 20 tools.
* `mcp_servers` - (Optional) JSON-encoded MCP servers array. Maximum 20 servers, names must be unique.
* `skills` - (Optional) JSON-encoded skills array. Maximum 20 skills.
* `multiagent` - (Optional) JSON-encoded multi-agent coordinator config.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.

## Attribute Reference

* `id` - Agent ID (`agt_...`).
* `version` - Optimistic-lock version, incremented on each update.
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by `workspace_id/agent_id`:

```shell
terraform import anthropic_wif_agent.example wrks_xxx/agt_yyy
```
