---
page_title: "anthropic Provider - Elmanuel1/anthropic-managed-agents"
description: |-
  Terraform provider for managing Anthropic platform resources. Supports Admin API key, workspace API key, and Workload Identity Federation (WIF) authentication — each resource validates only what it needs.
---

# anthropic Provider

Manage Anthropic platform resources using Terraform. All provider attributes are optional — each resource validates only the credentials it needs at operation time.

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
  # All attributes optional. Each falls back to the corresponding environment variable.
  admin_api_key      = var.anthropic_admin_api_key  # or ANTHROPIC_ADMIN_API_KEY
  api_key            = var.anthropic_api_key         # or ANTHROPIC_API_KEY
  federation_rule_id = var.anthropic_federation_rule_id  # or ANTHROPIC_FEDERATION_RULE_ID
  organization_id    = var.anthropic_organization_id     # or ANTHROPIC_ORGANIZATION_ID
  service_account_id = var.anthropic_service_account_id  # or ANTHROPIC_SERVICE_ACCOUNT_ID
}
```

Use [provider aliases](https://developer.hashicorp.com/terraform/language/providers/configuration#alias-multiple-provider-configurations) to configure multiple workspaces or authentication methods in the same root module:

```terraform
provider "anthropic" {
  alias         = "management"
  admin_api_key = var.anthropic_admin_api_key
}

provider "anthropic" {
  alias              = "wif"
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
}

provider "anthropic" {
  alias   = "workspace_a"
  api_key = var.workspace_a_api_key
}
```

## Schema

| Attribute | Type | Env var fallback | Required for |
|---|---|---|---|
| `admin_api_key` | String, sensitive | `ANTHROPIC_ADMIN_API_KEY` | `anthropic_workspace`, `anthropic_memory_store` |
| `api_key` | String, sensitive | `ANTHROPIC_API_KEY` | `anthropic_agent` |
| `federation_rule_id` | String | `ANTHROPIC_FEDERATION_RULE_ID` | `anthropic_wif_*` resources |
| `organization_id` | String | `ANTHROPIC_ORGANIZATION_ID` | `anthropic_wif_*` resources |
| `service_account_id` | String | `ANTHROPIC_SERVICE_ACCOUNT_ID` | `anthropic_wif_*` resources |

The OIDC JWT (`TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` or `TFC_WORKLOAD_IDENTITY_TOKEN`) is read from the environment only — Terraform Cloud injects it automatically per run.

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

```terraform
terraform {
  required_providers {
    anthropic = {
      source  = "Elmanuel1/anthropic-managed-agents"
      version = "~> 0.0"
    }
  }
}

provider "anthropic" {}

resource "anthropic_workspace" "example" {
  name = "my-workspace"
}

resource "anthropic_wif_agent" "example" {
  workspace_id = anthropic_workspace.example.id
  name         = "my-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."
}
```

## Resources

| Resource | Auth | Description |
|---|---|---|
| [`anthropic_workspace`](resources/workspace.md) | `admin_api_key` | Anthropic workspace |
| [`anthropic_memory_store`](resources/memory_store.md) | `admin_api_key` | Memory store for agent persistence |
| [`anthropic_agent`](resources/agent.md) | `api_key` | Agent managed with a workspace API key |
| [`anthropic_wif_agent`](resources/wif_agent.md) | WIF | Agent managed via TFC OIDC federation |
| [`anthropic_wif_environment`](resources/wif_environment.md) | WIF | Execution environment for agents |
| [`anthropic_wif_vault`](resources/wif_vault.md) | WIF | Vault for storing credentials |
| [`anthropic_wif_vault_credential`](resources/wif_vault_credential.md) | WIF | Credential stored in a vault |

## Guides

- [Authentication & Debugging](guides/authentication.md): how WIF token exchange works, reading authentication events, and fixing common failures.
