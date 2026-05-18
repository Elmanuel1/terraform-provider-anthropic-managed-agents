---
page_title: "anthropic Provider - Elmanuel1/anthropic-managed-agents"
description: |-
  Terraform provider for managing Anthropic platform resources. Supports Admin API key, Workload Identity Federation (WIF), and workspace API key authentication.
---

# anthropic Provider

Manage Anthropic platform resources using Terraform. All provider attributes are optional; each resource validates only the credentials it needs at operation time.

## Provider Configuration

```terraform
terraform {
  required_providers {
    anthropic = {
      source  = "Elmanuel1/anthropic-managed-agents"
      version = "~> 0.0"
    }
  }
}

provider "anthropic" {
  admin_api_key      = var.anthropic_admin_api_key
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
  workspace_api_key  = var.anthropic_workspace_api_key
}
```

The OIDC JWT (`TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` or `TFC_WORKLOAD_IDENTITY_TOKEN`) is read from the environment only — Terraform Cloud injects it automatically per run.

## Schema

| Attribute | Type | Required for |
|---|---|---|
| `admin_api_key` | String, sensitive | `anthropic_workspace`, `anthropic_memory_store` |
| `workspace_api_key` | String, sensitive | `anthropic_agent` (when not using WIF) |
| `federation_rule_id` | String | `anthropic_agent`, `anthropic_environment`, `anthropic_vault`, `anthropic_vault_credential` (WIF) |
| `organization_id` | String | `anthropic_agent`, `anthropic_environment`, `anthropic_vault`, `anthropic_vault_credential` (WIF) |
| `service_account_id` | String | `anthropic_agent`, `anthropic_environment`, `anthropic_vault`, `anthropic_vault_credential` (WIF) |

### Anthropic Console Setup

1. **Workload Identity Issuer**: Console → Settings → Workload Identity → Create issuer
   - Issuer URL: `https://app.terraform.io`
   - JWKS source: `discovery`
   - Max token lifetime: `2h`

2. **Service Account**: Console → Settings → Service Accounts → Create
   - Assign `Workspace Developer` role on every workspace this service account needs to manage resources in

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

See the [Authentication Events guide](guides/authentication.md) for debugging token exchange failures.

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

## Resources

| Resource | Auth | Description |
|---|---|---|
| [`anthropic_workspace`](resources/workspace.md) | `admin_api_key` | Anthropic workspace |
| [`anthropic_memory_store`](resources/memory_store.md) | `admin_api_key` | Memory store for agent persistence |
| [`anthropic_agent`](resources/agent.md) | WIF or `workspace_api_key` | Anthropic agent |
| [`anthropic_environment`](resources/environment.md) | WIF | Execution environment for agents |
| [`anthropic_vault`](resources/vault.md) | WIF | Vault for storing credentials |
| [`anthropic_vault_credential`](resources/vault_credential.md) | WIF | Credential stored in a vault |

## Guides

- [Authentication & Debugging](guides/authentication.md): how WIF token exchange works, reading authentication events, and fixing common failures.
