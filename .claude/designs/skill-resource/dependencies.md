# anthropic_skill Resource — Dependency Ledger

| # | Dependency | Assumed behavior we relied on | Evidence | Status |
|---|-----------|-------------------------------|----------|--------|
| 1 | `/v1/skills` endpoint | POST/GET/DELETE exist, multipart upload | Verified via live API calls during implementation | ✅ VERIFIED |
| 2 | CRUD ops | Create, Read, Delete — no Update on skill itself; content changes go through versions | Confirmed: no PATCH/PUT on skill; `POST /v1/skills/{id}/versions` is the update path | ✅ VERIFIED |
| 3 | Response shape | `id`, `display_title` (nullable pointer), `created_at`, `updated_at` present on create response | Live API response — `created_at` always present, no need for fallback Read after Create | ✅ VERIFIED |
| 4 | Auth | Workspace API key required (not admin key) | HTTP 404 with admin key; HTTP 200 with workspace key. SDK confirms `beta.skills` is workspace-scoped. | ✅ VERIFIED — design was wrong (assumed admin key) |
| 5 | Beta header | `anthropic-beta: skills-2025-10-02` required on all requests; `?beta=true` also required on DELETE and list-versions | Live API — 400 without header; delete fails without query param | ✅ VERIFIED |
| 6 | File field name | Multipart field must be `files[]` (bracket notation) | HTTP 400 "No files provided" with `files`; 200 with `files[]` | ✅ VERIFIED — design was wrong (assumed `files`) |
| 7 | File path prefix | Every file path must be prefixed with the skill `name` from SKILL.md frontmatter (e.g. `my-skill/SKILL.md`) | HTTP 400 "SKILL.md must be in top-level folder" without prefix; resolved by parsing `name:` field from YAML frontmatter | ✅ VERIFIED — not anticipated in design |
| 8 | SKILL.md format | Must have YAML frontmatter (`---`) with a `name` field | HTTP 400 "SKILL.md must start with YAML frontmatter" | ✅ VERIFIED — not anticipated in design |
| 9 | Versioning model | `POST /v1/skills/{id}/versions` creates new version | Confirmed via live API | ✅ VERIFIED |
| 10 | Version ID format | Delete version uses numeric `version` field (Unix timestamp string), not the `id` field | HTTP 400 "Invalid version format" with `id`; resolved by using `version` field from list response | ✅ VERIFIED — design was wrong (assumed `id`) |
| 11 | Delete order | Must delete all versions before deleting the skill | HTTP 400 "Cannot delete skill with existing versions" on direct delete | ✅ VERIFIED — not anticipated in design |
| 12 | WIF + skills | WIF is not applicable — `/v1/skills` has no workspace scope; `resolveWorkspaceCredentials` called with empty `workspace_id`, so WIF path is never taken | Code path: `workspaceID = ""` → WIF branch skipped → falls through to workspace API key | ✅ VERIFIED |
| 13 | `WithBeta` as sole beta mechanism | `WithBeta(creds, beta)` wraps any credential type and overrides the beta header after auth | Implemented and tested: workspace API key + WIF both work via `WithBeta`; inline `Beta` fields removed from `WorkspaceAPIKey`, `AdminAPIKey`, and `WIFBearer` | ✅ VERIFIED |
| 14 | WIF local testing | Anthropic token exchange (`/v1/oauth/token`) fetches JWKS from the issuer's OIDC discovery endpoint to verify JWT signatures | Confirmed: ngrok-exposed local OIDC server JWKS fetched successfully; `jwt_kid_not_in_jwks`, `jwt_audience_mismatch`, `sa_not_in_workspace` all surfaced with clear reason codes in dashboard | ✅ VERIFIED |
