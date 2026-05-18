---
page_title: "anthropic Provider - Elmanuel1/anthropic-managed-agents"
description: |-
  Terraform provider for managing Anthropic platform resources. Supports Admin API key, Workload Identity Federation (WIF), and workspace API key authentication.
---

# anthropic Provider

Manage Anthropic platform resources using Terraform.

## Authentication Modes

This provider supports three credential types. Each resource uses exactly one:

| Auth mode | When to use | Credentials needed |
|---|---|---|
| **Admin API key** | Managing workspaces and memory stores | `admin_api_key` |
| **WIF** | Managing agents, environments, vaults, and vault credentials from Terraform Cloud | `federation_rule_id` + `organization_id` + `service_account_id` + TFC-injected OIDC JWT |
| **Workspace API key** | Managing agents outside of Terraform Cloud (no WIF) | `workspace_api_key` |

For `anthropic_agent`, WIF and workspace API key are both supported. **WIF takes precedence when both are configured.**

All provider attributes are optional — each resource validates only the credentials it needs at apply time.

## Provider Configuration

```terraform
terraform {
  required_providers {
    anthropic = {
      source  = "Elmanuel1/anthropic-managed-agents"
      version = "~> 0.1"
    }
  }
}

provider "anthropic" {
  # Admin API key — required for anthropic_workspace and anthropic_memory_store
  admin_api_key = var.anthropic_admin_api_key

  # WIF — required for anthropic_agent (WIF mode), anthropic_environment, anthropic_vault, anthropic_vault_credential
  # The OIDC JWT is injected automatically by Terraform Cloud; no provider attribute needed for it
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id

  # Workspace API key — alternative to WIF for anthropic_agent only
  workspace_api_key = var.anthropic_workspace_api_key
}
```

## Schema

| Attribute | Type | Auth mode |
|---|---|---|
| `admin_api_key` | String, sensitive | Admin API key |
| `workspace_api_key` | String, sensitive | Workspace API key |
| `federation_rule_id` | String | WIF |
| `organization_id` | String | WIF |
| `service_account_id` | String | WIF |

The WIF OIDC JWT (`TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` or `TFC_WORKLOAD_IDENTITY_TOKEN`) is read from the environment only — Terraform Cloud injects it automatically per run.

## Example Usage

### Admin API key + WIF (full setup)

```terraform
provider "anthropic" {
  admin_api_key      = var.anthropic_admin_api_key
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

### Workspace API key only (no TFC OIDC)

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

## Resources

| Resource | Auth mode | Description |
|---|---|---|
| [`anthropic_workspace`](resources/workspace.md) | Admin API key | Anthropic workspace |
| [`anthropic_memory_store`](resources/memory_store.md) | Admin API key | Memory store for agent persistence |
| [`anthropic_agent`](resources/agent.md) | WIF or workspace API key | Anthropic agent |
| [`anthropic_environment`](resources/environment.md) | WIF | Execution environment for agents |
| [`anthropic_vault`](resources/vault.md) | WIF | Vault for storing credentials |
| [`anthropic_vault_credential`](resources/vault_credential.md) | WIF | Credential stored in a vault |

## WIF Console Setup

Required when using WIF authentication:

1. **Workload Identity Issuer**: Console → Settings → Workload Identity → Create issuer
   - Issuer URL: `https://app.terraform.io`
   - JWKS source: `discovery`
   - Max token lifetime: `2h`

2. **Service Account**: Console → Settings → Service Accounts → Create
   - Assign `Workspace Developer` role on every workspace this service account manages

3. **Federation Rule**: Console → Settings → Federation Rules → Create
   - Issuer: the issuer from step 1
   - Target: the service account from step 2
   - Scope: `workspace:developer`
   - Token lifetime: `2h`
   - Match type: `CEL`
   - CEL condition (replace `<tfc-org>`, `<tfc-project>`, `<tfc-workspace>`):

   ```cel
   claims.sub.matches("^organization:<tfc-org>:project:<tfc-project>:workspace:<tfc-workspace>:run_phase:(plan|apply)$")
   ```

See the [Authentication & Debugging guide](guides/authentication.md) for troubleshooting WIF token exchange failures.
