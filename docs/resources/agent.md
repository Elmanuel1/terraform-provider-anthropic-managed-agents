---
page_title: "anthropic: anthropic_agent"
subcategory: ""
description: |-
  Manages an Anthropic agent.
---

# Resource: anthropic_agent

Manages an Anthropic agent.

Supports two authentication modes, controlled by what is set in the **provider block**:

| Mode | Provider attributes required | `workspace_id` |
|---|---|---|
| WIF | `federation_rule_id`, `organization_id`, `service_account_id` | Required |
| Workspace API key | `workspace_api_key` | Not needed |

When both are configured, WIF takes precedence.

For Terraform Cloud WIF setup and debugging token exchange failures, see the [Authentication guide](../guides/authentication.md).

## Example Usage

### WIF authentication

```terraform
provider "anthropic" {
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
}

resource "anthropic_workspace" "example" {
  name = "my-workspace"
}

resource "anthropic_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "my-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."
}
```

### Workspace API key authentication

```terraform
provider "anthropic" {
  workspace_api_key = var.anthropic_workspace_api_key
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

### Agent with Anthropic skills

```terraform
resource "anthropic_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "data-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a data analysis assistant."

  skills = jsonencode([
    { "type" = "anthropic", "skill_id" = "xlsx" },
    { "type" = "anthropic", "skill_id" = "web_search" }
  ])
}
```

### Multi-agent coordinator

```terraform
resource "anthropic_agent" "worker" {
  workspace_id = anthropic_workspace.example.id
  name         = "worker"
  model        = "claude-sonnet-4-6"
  system       = "You are a worker agent."
}

resource "anthropic_agent" "coordinator" {
  workspace_id = anthropic_workspace.example.id
  name         = "coordinator"
  model        = "claude-opus-4-7"

  multiagent = jsonencode({
    type   = "coordinator"
    agents = [anthropic_agent.worker.id]
  })
}
```

## Argument Reference

* `workspace_id` - (Optional, Forces new resource) Workspace ID. Required when using WIF authentication.
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

WIF (workspace_id known):

```shell
terraform import anthropic_agent.example wrks_xxx/agt_yyy
```

Workspace API key (workspace_id not needed):

```shell
terraform import anthropic_agent.example agt_yyy
```
