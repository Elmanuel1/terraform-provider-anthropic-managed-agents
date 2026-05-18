---
page_title: "Provider: anthropic"
description: |-
  Use the Anthropic provider to manage Anthropic platform resources. The provider supports Admin API key, Workload Identity Federation (WIF), and workspace API key authentication.
---

# Anthropic Provider

## Example Usage

### Admin API key

Used by `anthropic_workspace` and `anthropic_memory_store`.

```terraform
provider "anthropic" {
  admin_api_key = var.anthropic_admin_api_key
}
```

### Workload Identity Federation (WIF)

Used by `anthropic_agent`, `anthropic_environment`, `anthropic_vault`, and `anthropic_vault_credential`. Requires Terraform Cloud — the OIDC JWT is injected automatically per run.

```terraform
provider "anthropic" {
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
}
```

See the [Authentication & Debugging guide](guides/authentication.md) for WIF setup and troubleshooting.

### Workspace API key

Used by `anthropic_agent` when running outside Terraform Cloud. When both WIF and workspace API key are configured, WIF takes precedence.

```terraform
provider "anthropic" {
  workspace_api_key = var.anthropic_workspace_api_key
}
```

### All credentials

```terraform
provider "anthropic" {
  admin_api_key      = var.anthropic_admin_api_key
  federation_rule_id = var.anthropic_federation_rule_id
  organization_id    = var.anthropic_organization_id
  service_account_id = var.anthropic_service_account_id
  workspace_api_key  = var.anthropic_workspace_api_key
}
```

## Schema

### Optional

- `admin_api_key` (String, Sensitive) Anthropic Admin API key (`sk-ant-admin-...`).
- `workspace_api_key` (String, Sensitive) Anthropic workspace API key (`sk-ant-api03-...`).
- `federation_rule_id` (String) Federation rule ID (`fdrl_...`).
- `organization_id` (String) Anthropic organization UUID.
- `service_account_id` (String) Service account ID (`svac_...`).
