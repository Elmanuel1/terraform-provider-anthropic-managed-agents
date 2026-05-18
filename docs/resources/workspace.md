---
page_title: "anthropic: anthropic_workspace"
subcategory: ""
description: |-
  Manages an Anthropic workspace.
---

# Resource: anthropic_workspace

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

This resource supports the following arguments:

* `name` - (Required) Workspace name as it appears in the Anthropic Console.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Workspace ID (`wrks_...`).
* `created_at` - ISO 8601 creation timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by workspace name (resolved to ID at import time):

```shell
terraform import anthropic_workspace.example my-workspace
```
