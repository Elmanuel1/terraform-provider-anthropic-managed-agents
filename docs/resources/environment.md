---
page_title: "anthropic: anthropic_environment"
subcategory: ""
description: |-
  Manages an Anthropic cloud environment for agent sessions.
---

# Resource: anthropic_environment

Manages an Anthropic cloud execution environment. Environments define the runtime configuration for agent sessions: networking policy, pre-installed packages, and MCP server access.

Authenticates via WIF bearer token scoped to the `workspace_id`.

On destroy the environment is archived by default. Set `force_delete = true` to permanently delete it.

## Example Usage

### Unrestricted environment

```terraform
resource "anthropic_environment" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "default-env"
}
```

### Limited networking with packages

```terraform
resource "anthropic_environment" "example" {
  workspace_id    = anthropic_workspace.example.id
  name            = "python-env"
  networking_type = "limited"

  allowed_hosts          = ["pypi.org", "files.pythonhosted.org"]
  allow_package_managers = true

  packages = jsonencode({
    pip = ["pandas", "numpy", "requests"]
  })

  metadata = {
    team = "data-science"
  }
}
```

## Argument Reference

This resource supports the following arguments:

* `workspace_id` - (Required, Forces new resource) Workspace ID.
* `name` - (Required) Environment name.
* `description` - (Optional) Human-readable description.
* `networking_type` - (Optional) `unrestricted` (default) or `limited`.
* `allowed_hosts` - (Optional) Allowed outbound hostnames. Only applies when `networking_type = "limited"`.
* `allow_mcp_servers` - (Optional) Allow MCP server network access. Default `false`. Only applies when `networking_type = "limited"`.
* `allow_package_managers` - (Optional) Allow package manager network access (PyPI, npm, etc). Default `false`. Only applies when `networking_type = "limited"`.
* `packages` - (Optional) JSON-encoded packages to pre-install. Supported managers: `apt`, `cargo`, `gem`, `go`, `npm`, `pip`.
* `metadata` - (Optional) Map of arbitrary string key-value pairs.
* `force_delete` - (Optional) When `true`, permanently deletes on destroy. Default `false` (archives).

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Environment ID (`env_...`).
* `created_at` - ISO 8601 creation timestamp.
* `updated_at` - ISO 8601 last-updated timestamp.
* `archived_at` - ISO 8601 archival timestamp, or null if active.

## Import

Import by `workspace_id/environment_id`:

```shell
terraform import anthropic_environment.example wrks_xxx/env_yyy
```
