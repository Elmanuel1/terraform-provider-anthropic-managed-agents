# terraform-provider-anthropic-managed-agents

Terraform provider for managing Anthropic platform resources.

Registry: [registry.terraform.io/providers/Elmanuel1/anthropic-managed-agents](https://registry.terraform.io/providers/Elmanuel1/anthropic-managed-agents/latest)

## Resources

| Resource | Auth | Description |
|---|---|---|
| `anthropic_workspace` | `admin_api_key` | Anthropic workspace |
| `anthropic_memory_store` | `admin_api_key` | Memory store for agent persistence |
| `anthropic_agent` | WIF or `workspace_api_key` | Anthropic agent |
| `anthropic_environment` | WIF | Execution environment for agents |
| `anthropic_vault` | WIF | Vault for storing credentials |
| `anthropic_vault_credential` | WIF | MCP server credential in a vault |

## Quick Start

### WIF authentication

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

## Provider Configuration

| Attribute | Description | Required for |
|---|---|---|
| `admin_api_key` | Admin API key (`sk-ant-admin-...`) | `anthropic_workspace`, `anthropic_memory_store` |
| `workspace_api_key` | Workspace API key (`sk-ant-api03-...`) | `anthropic_agent` (non-WIF) |
| `federation_rule_id` | Federation rule ID (`fdrl_...`) | WIF resources |
| `organization_id` | Organization UUID | WIF resources |
| `service_account_id` | Service account ID (`svac_...`) | WIF resources |

The WIF token (`TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` or `TFC_WORKLOAD_IDENTITY_TOKEN`) is read from the environment — Terraform Cloud injects it automatically.

## Anthropic Console Setup

1. **Workload Identity Issuer**: Console → Settings → Workload Identity → Create issuer
   - Issuer URL: `https://app.terraform.io` | JWKS source: `discovery` | Max token lifetime: `2h`

2. **Service Account**: Console → Settings → Service Accounts → Create
   - Assign `Workspace Developer` on each workspace this account manages

3. **Federation Rule**: Console → Settings → Federation Rules → Create
   - Target: service account from step 2 | Scope: `workspace:developer` | Token lifetime: `2h`
   - CEL condition:
     ```cel
     claims.sub.matches("^organization:<tfc-org>:project:<tfc-project>:workspace:<tfc-workspace>:run_phase:(plan|apply)$")
     ```

## Local Development

```bash
go build -o terraform-provider-anthropic-managed-agents .
```

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "Elmanuel1/anthropic-managed-agents" = "/path/to/provider/binary"
  }
  direct {}
}
```
