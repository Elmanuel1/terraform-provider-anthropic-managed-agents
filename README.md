# terraform-provider-anthropic-managed-agents

Terraform provider for managing Anthropic platform resources using Workload Identity Federation (WIF) via Terraform Cloud OIDC.

Registry: [registry.terraform.io/providers/Elmanuel1/anthropic-managed-agents](https://registry.terraform.io/providers/Elmanuel1/anthropic-managed-agents/latest)

## Resources

| Resource | Auth | Description |
|---|---|---|
| `anthropic_workspace` | Admin API key | Anthropic workspace |
| `anthropic_wif_agent` | WIF | Agent with model, tools, and skills |
| `anthropic_wif_environment` | WIF | Execution environment for agents |
| `anthropic_wif_vault` | WIF | Vault for storing credentials |
| `anthropic_wif_vault_credential` | WIF | MCP server credential in a vault |
| `anthropic_memory_store` | Admin API key | Memory store for agent persistence |

## Quick Start

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

## Authentication

### Environment Variables

| Variable | Description | Required |
|---|---|---|
| `ANTHROPIC_ADMIN_API_KEY` | Admin API key (`sk-ant-admin-...`) | Always |
| `ANTHROPIC_FEDERATION_RULE_ID` | Federation rule ID (`fdrl_...`) | WIF resources |
| `ANTHROPIC_ORGANIZATION_ID` | Organization UUID | WIF resources |
| `ANTHROPIC_SERVICE_ACCOUNT_ID` | Service account ID (`svac_...`) | WIF resources |
| `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` | TFC-injected OIDC JWT (set `TFC_WORKLOAD_IDENTITY_AUDIENCE_ANTHROPIC=https://api.anthropic.com`) | WIF resources |
| `TFC_WORKLOAD_IDENTITY_TOKEN` | Generic TFC OIDC JWT (set `TFC_WORKLOAD_IDENTITY_AUDIENCE=https://api.anthropic.com`) | WIF resources |

### Anthropic Console Setup

1. **Workload Identity Issuer**
   - Console → Settings → Workload Identity → Create issuer
   - Issuer URL: `https://app.terraform.io` | JWKS source: `discovery` | Max token lifetime: `2h`

2. **Service Account**
   - Console → Settings → Service Accounts → Create
   - Assign `Workspace Developer` on each workspace this account manages

3. **Federation Rule**
   - Console → Settings → Federation Rules → Create
   - Match type: `CEL`
   - CEL condition:
     ```cel
     claims.sub.matches("^organization:<tfc-org>:project:<tfc-project>:workspace:<tfc-workspace>:run_phase:(plan|apply)$")
     ```
   - Target: service account from step 2 | Scope: `workspace:developer` | Token lifetime: `2h`

## Local Development

```bash
go build -o terraform-provider-anthropic-managed-agents .

# ~/.terraformrc
cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "Elmanuel1/anthropic-managed-agents" = "/path/to/provider/binary"
  }
  direct {}
}
EOF

export ANTHROPIC_ADMIN_API_KEY="sk-ant-admin-..."
export ANTHROPIC_FEDERATION_RULE_ID="fdrl_..."
export ANTHROPIC_ORGANIZATION_ID="00000000-..."
export ANTHROPIC_SERVICE_ACCOUNT_ID="svac_..."
export TFC_WORKLOAD_IDENTITY_TOKEN="<jwt>"

terraform plan
```
