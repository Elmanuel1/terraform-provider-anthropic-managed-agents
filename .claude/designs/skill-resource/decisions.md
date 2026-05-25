# anthropic_skill Resource — Design Decisions

## Decision Summary

| Decision | Resolved To |
|---|---|
| API endpoint | `POST/GET/DELETE /v1/skills`, `POST /v1/skills/{id}/versions`. Beta header `anthropic-beta: skills-2025-10-02`. Delete and list-versions also require `?beta=true` query param. |
| Auth | Workspace API key (not admin key). Uses `resolveWorkspaceCredentials` + `WithBeta(creds, auth.SkillsBeta)`. WIF not supported for skills (no workspace_id scope on the endpoint). |
| File content input | Directory path (`source_dir`). Provider walks all files, uploads as multipart `files[]` fields. Validates SKILL.md exists at root. Each file path is prefixed with the skill name parsed from SKILL.md YAML frontmatter (e.g. `my-skill/SKILL.md`). |
| Versioning | Auto-create new version when `source_hash` changes. No separate resource. |
| Change detection | SHA-256 hash of all files in `source_dir` (sorted by relative path, path + content both hashed). Stored as `source_hash` in state. Recomputed on every plan via `sourceHashPlanModifier`. |
| Destroy | Delete all versions first (list by numeric timestamp `version` field, delete each), then delete skill. Both steps require `?beta=true`. |
| `display_title` | Required. Triggers force-replace (`RequiresReplace` plan modifier) — no update endpoint on skill itself. |
| Data source | Yes — `anthropic_skill` data source reads by ID. |
| Import | By skill ID (`skill_...`). `source_dir` and `source_hash` are left null on import (locally managed). |
| Beta injection | `WithBeta` wrapper — single mechanism across all credential types. No inline `Beta` field on credential structs. |

## Resource Schema

```hcl
resource "anthropic_skill" "example" {
  display_title = "My Skill"       # Required string — change forces replacement
  source_dir    = "./my-skill"     # Required — directory containing SKILL.md at root
}
```

Computed attributes: `id`, `source_hash`, `created_at`, `updated_at`.

Not in schema: `latest_version`, `source` — internal API fields not needed in Terraform state.

## Implementation Notes

- **Auth**: `resolveWorkspaceCredentials` returns bare creds; `skillClient()` helper wraps with `auth.WithBeta(creds, auth.SkillsBeta)`. Consistent across all four CRUD methods.
- **Client**: `internal/client/skill_client.go`. Methods: `Create`, `Read`, `Delete`, `CreateVersion`, `listVersions`.
- **File path prefix**: `parseSkillName()` reads YAML frontmatter from SKILL.md to get the `name` field. All files are uploaded as `skillname/relpath` — the API rejects uploads where paths don't start with the declared skill name.
- **Multipart field name**: `files[]` (not `files`) — API requires the bracket notation.
- **Version IDs**: Delete uses the numeric `version` field (Unix timestamp string), not the `id` field, from the versions list response.
- **Hash validation**: `computeSourceHash` validates non-empty dir and SKILL.md presence before computing — surfaces errors at plan time via `sourceHashPlanModifier`.
- **State preservation on Read**: `source_dir` and `source_hash` are preserved from prior state on Read — the API has no knowledge of local paths.
- **WIF local testing**: `cmd/local-oidc` provides a local OIDC server + ngrok workflow. See `docs/wif-local-testing.md`.
