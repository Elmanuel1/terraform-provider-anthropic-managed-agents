---
page_title: "anthropic-wif_environment Resource"
description: |-
  Manages an Anthropic cloud environment for agent sessions.
---

# anthropic-wif_environment

Manages an Anthropic cloud execution environment. Environments define the runtime configuration for agent sessions: networking policy, pre-installed packages, and MCP server access.

Authenticates via WIF bearer token scoped to the `workspace_id`.

On destroy the environment is **archived** by default. Set `force_delete = true` to permanently delete it.

## Example Usage

### Unrestricted environment

```terraform
resource "anthropic-wif_environment" "example" {
  workspace_id = anthropic-wif_workspace.example.id
  name         = "default-env"
}
```

### Limited networking with packages

```terraform
resource "anthropic-wif_environment" "example" {
  workspace_id    = anthropic-wif_workspace.example.id
  name            = "python-env"
  networking_type = "limited"

  allowed_hosts           = ["pypi.org", "files.pythonhosted.org"]
  allow_package_managers  = true

  packages = jsonencode({
    pip = ["pandas", "numpy", "requests"]
  })

  metadata = {
    team = "data-science"
  }
}
```

## Import

Import by `workspace_id/environment_id`:

```shell
terraform import anthropic-wif_environment.example wrks_xxx/env_yyy
```

## Argument Reference

| Argument | Type | Required | Description |
|---|---|---|---|
| `workspace_id` | string | Yes | Workspace ID. Changing this forces a new resource. |
| `name` | string | Yes | Environment name. |
| `description` | string | No | Human-readable description. |
| `networking_type` | string | No | `unrestricted` (default) or `limited`. |
| `allowed_hosts` | list(string) | No | Allowed outbound hostnames. Only applies when `networking_type = "limited"`. |
| `allow_mcp_servers` | bool | No | Allow MCP server network access. Default `false`. Only applies when `networking_type = "limited"`. |
| `allow_package_managers` | bool | No | Allow package manager network access (PyPI, npm, etc). Default `false`. Only applies when `networking_type = "limited"`. |
| `packages` | string | No | JSON-encoded packages to pre-install. Supported managers: `apt`, `cargo`, `gem`, `go`, `npm`, `pip`. |
| `metadata` | map(string) | No | Arbitrary string key-value pairs. |
| `force_delete` | bool | No | When `true`, permanently deletes on destroy. Default `false` (archives). |

## Attribute Reference

| Attribute | Type | Description |
|---|---|---|
| `id` | string | Environment ID (`env_...`). |
| `created_at` | string | ISO 8601 creation timestamp. |
| `updated_at` | string | ISO 8601 last-updated timestamp. |
| `archived_at` | string | ISO 8601 archival timestamp, or null if active. |
