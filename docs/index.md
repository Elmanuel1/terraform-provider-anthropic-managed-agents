---
page_title: "Provider: anthropic"
description: |-
  Use the Anthropic provider to manage Anthropic platform resources. The provider supports Admin API key, Workload Identity Federation (WIF), and workspace API key authentication.
---

# Anthropic Provider

Use the Anthropic provider to manage Anthropic platform resources including workspaces, agents, environments, vaults, and memory stores.

## Authentication

The provider supports three authentication modes. Each resource uses exactly one:

- **Admin API key** — used by `anthropic_workspace` and `anthropic_memory_store`.
- **WIF (Workload Identity Federation)** — used by `anthropic_environment`, `anthropic_vault`, `anthropic_vault_credential`, and optionally `anthropic_agent`. Requires Terraform Cloud with OIDC token injection.
- **Workspace API key** — used by `anthropic_agent` when WIF is not available. When both WIF and workspace API key are configured, WIF takes precedence.

## Example Usage

### Admin API key and WIF

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

### Workspace API key

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

## Schema

### Optional

- `admin_api_key` (String, Sensitive) Admin API key (`sk-ant-admin-...`). Required for `anthropic_workspace` and `anthropic_memory_store`.
- `workspace_api_key` (String, Sensitive) Workspace API key (`sk-ant-api03-...`). Used for `anthropic_agent` authentication when WIF is not configured. When both are provided, WIF takes precedence.
- `federation_rule_id` (String) Federation rule ID (`fdrl_...`). Required for WIF-authenticated resources.
- `organization_id` (String) Anthropic organization UUID. Required for WIF-authenticated resources.
- `service_account_id` (String) Service account ID (`svac_...`). Required for WIF-authenticated resources.

The WIF OIDC JWT is read from `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` (or `TFC_WORKLOAD_IDENTITY_TOKEN` as fallback). Terraform Cloud injects it automatically — no provider attribute is needed.

## WIF Setup

To use WIF authentication, configure the following in the Anthropic Console:

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

See the [Authentication & Debugging guide](guides/authentication.md) for troubleshooting token exchange failures.
