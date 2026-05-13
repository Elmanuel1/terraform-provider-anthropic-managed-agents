# terraform-provider-anthropic-wif (validator)

Minimal Terraform provider that mints Anthropic WIF tokens per workspace via TFC OIDC injection. Used to validate the full token exchange flow before building the real provider.

## What it does

On each `terraform plan` or `apply`:
1. Reads TFC-injected `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` as the assertion JWT
2. Resolves `workspace_name` → `workspace_id` via Admin API
3. POSTs to `POST /v1/oauth/token` (RFC 7523 jwt-bearer) per workspace
4. Outputs token prefix + expiry — proves exchange succeeded without exposing the token

## TFC workspace setup

### Environment variables (set in TFC workspace)

| Variable | Value | Sensitive |
|---|---|---|
| `TFC_WORKLOAD_IDENTITY_AUDIENCE_ANTHROPIC` | `https://api.anthropic.com` | No — triggers TFC to inject the JWT |
| `ANTHROPIC_ADMIN_API_KEY` | `sk-ant-...` | Yes — Admin API for workspace resolution only |
| `ANTHROPIC_FEDERATION_RULE_ID` | `fdrl_...` | Yes |
| `ANTHROPIC_ORGANIZATION_ID` | `00000000-...` | Yes |
| `ANTHROPIC_SERVICE_ACCOUNT_ID` | `svac_...` | Yes |

`TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` is injected automatically by TFC — do not set it manually.

### Anthropic Console setup

1. Console → Workload identity → Create issuer
   - Issuer URL: `https://app.terraform.io`
   - JWKS source: `discovery`
2. Console → Service accounts → Create service account
3. Console → Federation rules → Create rule
   - Audience: `https://api.anthropic.com`
   - Subject prefix: `organization:<your-tfc-org>:project:<project>:workspace:<workspace>:run_phase:`
   - Target: service account from step 2
   - Scope: `workspace:developer`

## Build and run locally (dev override)

```bash
cd tf-provider-wif-validator
go mod tidy
go build -o terraform-provider-anthropic-wif .

# ~/.terraformrc
cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/build4africa/anthropic-wif" = "/path/to/tf-provider-wif-validator"
  }
  direct {}
}
EOF

# Set env vars manually for local test (use ANTHROPIC_IDENTITY_TOKEN instead of TFC token)
export ANTHROPIC_IDENTITY_TOKEN="<manually minted JWT>"
export ANTHROPIC_FEDERATION_RULE_ID="fdrl_..."
export ANTHROPIC_ORGANIZATION_ID="00000000-..."
export ANTHROPIC_SERVICE_ACCOUNT_ID="svac_..."
export ANTHROPIC_ADMIN_API_KEY="sk-ant-..."

cd examples
terraform plan
```

## Run via TFC

```bash
cd examples
# Edit main.tf: set your TFC org + workspace name
terraform login
terraform init
terraform plan   # TFC runner injects TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC
```
