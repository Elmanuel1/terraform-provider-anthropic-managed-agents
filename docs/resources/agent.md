---
page_title: "anthropic: anthropic_agent"
subcategory: ""
description: |-
  Manages an Anthropic agent using a workspace API key.
---

# Resource: anthropic_agent

Manages an Anthropic agent. Authenticates with a workspace API key (`api_key` in the provider block or `ANTHROPIC_API_KEY`).

Use this resource when you have a static workspace API key. For Terraform Cloud workspaces using OIDC federation, use [`anthropic_wif_agent`](wif_agent.md) instead.

## Example Usage

### Single workspace

```terraform
provider "anthropic" {
  api_key = var.workspace_api_key
}

resource "anthropic_agent" "example" {
  name   = "my-agent"
  model  = "claude-sonnet-4-6"
  system = "You are a helpful assistant."
}
```

### Agent with tools and MCP servers

```terraform
resource "anthropic_agent" "example" {
  name        = "support-agent"
  model       = "claude-opus-4-7"
  model_speed = "standard"
  system      = "You are a customer support assistant."
  description = "Handles tier-1 support queries."

  tools = jsonencode([
    { "type" = "agent_toolset_20260401" }
  ])

  mcp_servers = jsonencode([
    {
      name = "helpdesk"
      type = "url"
      url  = "https://helpdesk.example.com/mcp"
    }
  ])

  metadata = {
    team = "support"
    env  = "production"
  }
}
```

### Multiple workspaces

```terraform
provider "anthropic" {
  alias   = "workspace_a"
  api_key = var.workspace_a_api_key
}

provider "anthropic" {
  alias   = "workspace_b"
  api_key = var.workspace_b_api_key
}

resource "anthropic_agent" "agent_a" {
  provider = anthropic.workspace_a
  name     = "agent-a"
  model    = "claude-sonnet-4-6"
}

resource "anthropic_agent" "agent_b" {
  provider = anthropic.workspace_b
  name     = "agent-b"
  model    = "claude-sonnet-4-6"
}
```

## Argument Reference

This resource supports the following arguments:

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

In addition to all arguments above, the following attributes are exported:

* `id` - Agent ID (`agt_...`).
* `version` - Optimistic-lock version, incremented on each update.
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by agent ID:

```shell
terraform import anthropic_agent.example agt_xxx
```
