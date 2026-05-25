# anthropic_skill Resource — Design Decisions

## Decision Summary

| Decision | Resolved To |
|---|---|
| API endpoint | `POST/GET/DELETE /v1/skills`, `POST /v1/skills/{id}/versions`. Beta header `anthropic-beta: skills-2025-10-02` + `?beta=true` query param. Admin API key auth. |
| File content input | Directory path (`source_dir`). Provider walks all files, uploads as multipart. Validates `SKILL.md` exists at root. |
| Versioning | Auto-create new version on content change. No separate resource. |
| Change detection | SHA-256 hash of all files in `source_dir` (sorted by path). Stored as `source_hash` in state. |
| Destroy | Hard delete. API errors surface directly to user. |
| Data source | Yes — `anthropic_skill` data source reads by ID. |
| Import | By skill ID (`skl_...`). |

## Resource Schema

```hcl
resource "anthropic_skill" "example" {
  display_title = "My Skill"       # Optional string
  source_dir    = "./my-skill"     # Required — directory containing SKILL.md at root
}
```

Computed attributes: `id`, `source_hash`, `latest_version`, `source`, `created_at`, `updated_at`.

## Implementation Notes

- Auth: `auth.AdminAPIKey` with beta `skills-2025-10-02`. `requireAdminKey` guard same as workspace resource.
- Client: `internal/client/skill_client.go` — new file. Methods: Create, Read, Delete, CreateVersion.
- File upload: walk `source_dir`, sort paths, build multipart request. Validate `SKILL.md` exists before upload.
- Hash: SHA-256 over concatenation of `<relpath>\n<content>` for each file sorted by relative path. Stored as hex string in `source_hash`.
- On plan: if `source_hash` in plan != state → `CreateVersion` in Update. If `display_title` changed and skill has no update endpoint → update display_title via… check if PATCH/POST on skill supports it. If not, force-recreate on display_title change.
- Data source: reads by `id`, returns same computed fields (no `source_dir` or `source_hash`).
- Register: add `NewSkillResource` and `NewSkillDataSource` to `provider.go`.
