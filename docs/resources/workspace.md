---
page_title: "anthropic_workspace Resource - anthropic"
subcategory: ""
description: |-
  Manages an Anthropic workspace.
---

# anthropic_workspace (Resource)

Manages an Anthropic workspace. Workspaces are the top-level organisational unit on the Anthropic platform. Agents, environments, vaults, and other resources are scoped to a workspace.

Authenticates with the Anthropic Admin API key (`ANTHROPIC_ADMIN_API_KEY`). WIF is not required for this resource.

On destroy the workspace is archived (soft-deleted). Anthropic does not expose a hard-delete endpoint for workspaces.

## Example Usage

```terraform
resource "anthropic_workspace" "example" {
  name = "my-workspace"
}
```

## Argument Reference

* `name` - (Required) Workspace name as it appears in the Anthropic Console.

## Attribute Reference

* `id` - Workspace ID (`wrks_...`).
* `created_at` - ISO 8601 creation timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by workspace name (resolved to ID at import time):

```shell
terraform import anthropic_workspace.example my-workspace
```
