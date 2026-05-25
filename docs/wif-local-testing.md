# Testing WIF Locally with ngrok

Workload Identity Federation (WIF) normally requires a Terraform Cloud run
because TFC injects the OIDC JWT automatically. This guide shows how to
replicate that using a local OIDC server exposed via ngrok.

## How it works

```
local-oidc server  ──(ngrok)──►  public URL
       │                              │
       │ signs JWT with private key   │ serves JWKS (public key)
       ▼                              ▼
 curl /mint               Anthropic token exchange
       │                  fetches JWKS, verifies JWT
       │                              │
       ▼                              ▼
TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC ──► WIF access token ──► API calls
```

## Prerequisites

- [ngrok](https://ngrok.com) installed (`brew install ngrok`)
- Go 1.21+
- An Anthropic service account and federation rule (one-time setup below)

## One-time setup

### 1. Start ngrok

```bash
ngrok http 8080
```

Note the `https://` URL (e.g. `https://abc123.ngrok-free.app`).

### 2. Start the local OIDC server

```bash
go run ./cmd/local-oidc --issuer https://abc123.ngrok-free.app
```

The server prints its JWKS and discovery URLs. Keep it running.

### 3. Create a WIF federation rule on Anthropic

In the Anthropic console, create a new federation rule:

| Field | Value |
|---|---|
| Issuer URL | `https://abc123.ngrok-free.app` |
| JWKS source | OIDC discovery |
| Discovery base URL | (leave blank) |
| Subject pattern | `local:*` |
| Audience | `https://api.anthropic.com` |

Save the `federation_rule_id`, `organization_id`, and `service_account_id`.

### 4. Add the service account to your target workspace

In the Anthropic console, open the workspace you want to manage and add
the service account as a member. WIF token exchange will fail with
`sa_not_in_workspace` if this step is skipped.

## Running terraform with WIF

Each `terraform apply` needs a fresh JWT. The `/mint` endpoint on the
running server signs it with the same key as the advertised JWKS.

```bash
# Mint a fresh token from the running server
export TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC=$(curl -s \
  'http://localhost:8080/mint?sub=local:test&aud=https://api.anthropic.com')

# Run terraform
TF_CLI_CONFIG_FILE=~/.terraformrc terraform apply \
  -var="federation_rule_id=fdrl_..." \
  -var="organization_id=<uuid>" \
  -var="service_account_id=svac_..."
```

The `/mint` endpoint accepts query params:

| Param | Default | Description |
|---|---|---|
| `sub` | `local:test` | Subject claim — must match the federation rule's subject pattern |
| `aud` | `https://api.anthropic.com` | Audience claim — must match the federation rule's audience |
| `ttl` | `10m` | Token lifetime (e.g. `30m`, `1h`) |

## Troubleshooting

| Dashboard reason | Fix |
|---|---|
| `jwt_kid_not_in_jwks` | You minted with a different server instance. Always use `curl /mint` against the running server, never `--mint` flag on a new process. |
| `jwt_audience_mismatch` | The `aud` in the JWT does not match the federation rule. Use `aud=https://api.anthropic.com`. |
| `sa_not_in_workspace` | Add the service account to the target workspace in the Anthropic console. |
| `jwt_subject_mismatch` | The `sub` in the JWT does not match the federation rule's subject pattern. Check the pattern (e.g. `local:*`). |

## Notes

- ngrok free tier URLs rotate on each restart. When that happens, create a
  new federation rule pointing to the new URL (or upgrade to a paid ngrok
  plan with a fixed domain).
- The local OIDC server generates a fresh RSA key pair on each start.
  After restarting it, Anthropic will fetch the new JWKS automatically on
  the next token exchange.
- Never commit API keys, federation rule IDs, or org IDs into source control.
  Pass them via environment variables or `-var` flags.
