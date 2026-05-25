# anthropic_skill Resource — Dependency Ledger

| # | Dependency | Assumed behavior | Evidence | Status |
|---|-----------|-----------------|----------|--------|
| 1 | `/v1/skills` endpoint | POST/GET/DELETE, multipart upload | Python SDK source: `anthropic.resources.beta.skills` | ✅ VERIFIED |
| 2 | CRUD ops | Create, Read, List, Delete — no Update on skill itself | SDK confirms no update method on skill; only versions.create for content changes | ✅ VERIFIED |
| 3 | Response shape | id, display_title, latest_version, source, created_at, updated_at | SDK response model (SkillResponse) | ✅ VERIFIED |
| 4 | Beta header | `anthropic-beta: skills-2025-10-02` + `?beta=true` query param required | SDK resource definition | ✅ VERIFIED |
| 5 | Org-level scope | `/v1/skills` has no workspace scope — admin API key only | SDK path, no workspace_id param anywhere | ✅ VERIFIED |
| 6 | File upload | Multipart/form-data, SKILL.md required at root, multiple files supported | SDK `files: Sequence[FileTypes]`, API docs | ✅ VERIFIED |
| 7 | Versioning model | `POST /v1/skills/{id}/versions` creates new version; API assigns Unix epoch timestamp as version ID; `latest_version` field on skill always reflects most recent | SDK `versions.create()` and SkillVersion response shape | ✅ VERIFIED |
