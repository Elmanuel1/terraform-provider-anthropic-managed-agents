---
page_title: "Authentication & Debugging"
description: |-
  How WIF token exchange works, how to read authentication events in the Anthropic Console, and how to fix common failures.
---

# Authentication & Debugging

## How WIF Token Exchange Works

Every workspace-scoped resource (agent, environment, vault, vault credential, memory store) authenticates via a short-lived bearer token obtained through Workload Identity Federation (WIF).

The flow on each Terraform run:

1. Terraform Cloud injects an OIDC JWT. Which variable it lands in depends on how your TFC workspace is configured: `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` when `TFC_WORKLOAD_IDENTITY_AUDIENCE_ANTHROPIC` is set, or `TFC_WORKLOAD_IDENTITY_TOKEN` when `TFC_WORKLOAD_IDENTITY_AUDIENCE` is set. The provider reads whichever is present.
2. The provider sends the JWT to `https://api.anthropic.com/v1/oauth/token` along with your `federation_rule_id`, `organization_id`, `service_account_id`, and `workspace_id`.
3. Anthropic validates the JWT against the registered issuer (`https://app.terraform.io`), evaluates the CEL condition on the federation rule, and if matched returns a workspace-scoped access token.
4. All subsequent API calls for that workspace use the access token as a `Bearer` token.

The provider caches the minted token per workspace ID so parallel resource creates in the same workspace share the single minted token rather than each attempting their own exchange.

## Authentication Events Panel

The Anthropic Console surfaces every token exchange attempt under **Settings → Workload Identity → Authentication events**. This is the fastest way to diagnose WIF failures.

### Reading the panel

| Column | What it shows |
|---|---|
| Time | When the exchange was attempted |
| Status | `Success` or `Failure` |
| Reason | Populated on failure (see table below) |
| Issuer | Which registered issuer the JWT came from |

Filter by **Issuer**, **Service account**, **Rule**, or **Outcome** to narrow results. The default range is the last hour; expand it if you are debugging an older run.

### Common failure reasons

| Reason | Cause | Fix |
|---|---|---|
| `jwks_unavailable` | Anthropic could not fetch the JWKS from `https://app.terraform.io/.well-known/jwks.json` at exchange time. Transient network issue on Anthropic's side. | Retry the TFC run. If it recurs, check the issuer's JWKS URL is reachable. |
| `jwt_expired` | The TFC OIDC JWT had already expired when the exchange was attempted. TFC JWTs are valid for a short window. | Ensure the provider is not caching a stale JWT across multiple runs. Each new run injects a fresh token. |
| `jti_reused` | Only possible when **JTI replay protection is enabled** on the issuer. With it disabled (the default), this error will not appear. If you enable JTI replay protection and see this, the provider's workspace-level token cache should prevent it; file a bug if it recurs. |
| `sub_mismatch` | The JWT's `sub` claim did not satisfy the CEL condition on the federation rule. | Check that the TFC org, project, and workspace names in the CEL regex exactly match what TFC injects. Names are case-sensitive. |
| `rule_not_found` | The `federation_rule_id` env var points to a rule that does not exist or has been deleted. | Verify `ANTHROPIC_FEDERATION_RULE_ID` is correct. |
| `service_account_not_found` | The `service_account_id` does not exist in the organization. | Verify `ANTHROPIC_SERVICE_ACCOUNT_ID` is correct. |
| `insufficient_scope` | The federation rule's OAuth scope does not grant the permissions needed. | Ensure the rule has `workspace:developer` scope. |
| `workspace_not_accessible` | The service account does not have `Workspace Developer` access to the target workspace. | In Console → Settings → Service Accounts, add the workspace with `Workspace Developer` role. |

### Reading a success event

A `Success` event with no reason means the token was minted. If a Terraform apply still fails with 401 after a successful exchange, the most likely cause is that the token expired before the apply finished. Request a longer lifetime (see below) or check the federation rule's token lifetime cap.

## Token Lifetime

The token lifetime is controlled by two settings that must both accommodate your longest expected run:

| Setting | Location | Recommended value |
|---|---|---|
| Issuer max token lifetime | Console → Settings → Workload Identity → edit issuer | `2h` |
| Federation rule token lifetime | Console → Settings → Federation Rules → edit rule | `2h` |

The server caps the issued lifetime at whichever of the two settings above is smaller. Set both to `2h` to cover the longest expected TFC runs.

## Debugging Checklist

1. Open Console → Settings → Workload Identity → Authentication events.
2. Set the range to cover your failed run.
3. Look for a `Failure` row with a reason. The table above maps each reason to a fix.
4. If all rows are `Success` but the apply still fails, check whether the token expired mid-run (increase issuer and rule lifetime to `2h`).
5. If there are no rows at all, the provider never attempted an exchange. Check that `ANTHROPIC_FEDERATION_RULE_ID`, `ANTHROPIC_ORGANIZATION_ID`, `ANTHROPIC_SERVICE_ACCOUNT_ID`, and `TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC` are all set on the TFC workspace.

## CEL Condition Reference

The federation rule uses a CEL condition to evaluate claims from the JWT. The most common pattern matches the `sub` claim using a regular expression.

A rule that allows both plan and apply phases for a single TFC workspace:

```cel
claims.sub.matches("^organization:my-org:project:my-project:workspace:my-workspace:run_phase:(plan|apply)$")
```

A rule scoped to apply phase only:

```cel
claims.sub.matches("^organization:my-org:project:my-project:workspace:my-workspace:run_phase:apply$")
```

A rule that covers all workspaces in one TFC project:

```cel
claims.sub.matches("^organization:my-org:project:my-project:workspace:[^:]+:run_phase:(plan|apply)$")
```

Names are case-sensitive and must exactly match the values shown in TFC.
