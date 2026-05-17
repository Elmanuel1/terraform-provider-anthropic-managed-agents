---
page_title: "anthropic-wif Provider"
description: |-
  Terraform provider for managing Anthropic workspaces, agents, environments, vaults, vault credentials, and memory stores using Workload Identity Federation (WIF) via TFC OIDC.
---

# anthropic-wif Provider

Manage Anthropic platform resources (workspaces, agents, environments, vaults, and vault credentials) using Terraform and Workload Identity Federation (WIF).

Workspace-scoped resources (agents, environments, vaults, vault credentials, memory stores) authenticate via a WIF bearer token minted from a TFC OIDC JWT. Workspace management uses the Anthropic Admin API key directly.

## Authentication

The provider requires:

| Environment Variable | Description | Required |
|---|---|---|
| `ANTHROPIC_ADMIN_API_KEY` | Anthropic Admin API key (`sk-ant-admin-...`) | Always |
| `ANTHROPIC_FEDERATION_RULE_ID` | Federation rule ID (`fdrl_...`) | WIF resources |
| `ANTHROPIC_ORGANIZATION_ID` | Anthropic organization UUID | WIF resources |
| `ANTHROPIC_SERVICE_ACCOUNT_ID` | Service account ID (`svac_...`) | WIF resources |
| `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` | Audience-specific OIDC JWT. Set `TFC_WORKLOAD_IDENTITY_AUDIENCE_ANTHROPIC=https://api.anthropic.com` on the TFC workspace and Terraform Cloud injects this automatically. | WIF resources |
| `TFC_WORKLOAD_IDENTITY_TOKEN` | Generic OIDC JWT. Set `TFC_WORKLOAD_IDENTITY_AUDIENCE=https://api.anthropic.com` on the TFC workspace if you use the single-audience slot. The provider reads this when `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` is absent. | WIF resources |

Use `TFC_WORKLOAD_IDENTITY_AUDIENCE_ANTHROPIC` (audience-specific) when the workspace already uses `TFC_WORKLOAD_IDENTITY_AUDIENCE` for another provider. Use the generic `TFC_WORKLOAD_IDENTITY_AUDIENCE` slot if Anthropic is the only workload identity consumer in that workspace.

### Anthropic Console Setup

1. **Workload Identity Issuer**: Console → Settings → Workload Identity → Create issuer
   - Issuer URL: `https://app.terraform.io`
   - JWKS source: `discovery`
   - Max token lifetime: `2h` (covers the longest TFC runs)

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

   Using `(plan|apply)` allows both plan and apply phases to exchange tokens. Restricting to `apply` only is also valid if you want tighter control.

See the [Authentication Events guide](guides/authentication.md) for debugging token exchange failures.

## Example Usage

```terraform
terraform {
  required_providers {
    anthropic-wif = {
      source  = "Elmanuel1/anthropic-wif"
      version = "~> 0.4"
    }
  }
}

provider "anthropic-wif" {}

resource "anthropic-wif_workspace" "example" {
  name = "my-workspace"
}

resource "anthropic-wif_agent" "example" {
  workspace_id = anthropic-wif_workspace.example.id
  name         = "my-agent"
  model        = "claude-sonnet-4-6"
  system       = "You are a helpful assistant."
}
```

## Resources

| Resource | Auth | Description |
|---|---|---|
| [`anthropic-wif_workspace`](resources/workspace.md) | Admin API key | Anthropic workspace |
| [`anthropic-wif_agent`](resources/agent.md) | WIF | Agent with model, tools, and skills |
| [`anthropic-wif_environment`](resources/environment.md) | WIF | Execution environment for agents |
| [`anthropic-wif_vault`](resources/vault.md) | WIF | Vault for storing credentials |
| [`anthropic-wif_vault_credential`](resources/vault_credential.md) | WIF | Credential stored in a vault |
| [`anthropic-wif_memory_store`](resources/memory_store.md) | WIF | Memory store for agent persistence |

## Guides

- [Authentication & Debugging](guides/authentication.md): how WIF token exchange works, reading authentication events, and fixing common failures.
