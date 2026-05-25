package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

const skillsPath = "/v1/skills"

type SkillResponse struct {
	ID            string  `json:"id"`
	DisplayTitle  *string `json:"display_title"`
	LatestVersion *string `json:"latest_version"`
	Source        string  `json:"source"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type SkillVersionResponse struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

type SkillClient struct {
	creds      auth.WorkspaceAPIKey
	httpClient *http.Client
}

func NewSkillClient(creds auth.WorkspaceAPIKey) *SkillClient {
	return &SkillClient{creds: creds, httpClient: defaultHTTPClient}
}

// Create uploads a new skill from sourceDir via multipart POST to /v1/skills.
func (c *SkillClient) Create(ctx context.Context, displayTitle, sourceDir string) (*SkillResponse, error) {
	body, contentType, err := buildMultipartBody(displayTitle, sourceDir)
	if err != nil {
		return nil, fmt.Errorf("creating skill: building multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, auth.BaseURL+skillsPath, body)
	if err != nil {
		return nil, fmt.Errorf("creating skill: building request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	if err := c.creds.Authenticate(ctx, req); err != nil {
		return nil, fmt.Errorf("creating skill: authenticating request: %w", err)
	}

	raw, status, err := c.doRaw(req)
	if err != nil {
		return nil, fmt.Errorf("creating skill: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating skill returned HTTP %d: %s", status, raw)
	}

	var s SkillResponse
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("creating skill: parsing response: %w", err)
	}
	if s.ID == "" {
		return nil, fmt.Errorf("creating skill: response did not include an id: %s", raw)
	}
	return &s, nil
}

// Read fetches a skill by ID from GET /v1/skills/{id}.
// Returns nil if the skill is not found (404).
func (c *SkillClient) Read(ctx context.Context, id string) (*SkillResponse, error) {
	raw, status, err := c.doJSON(ctx, http.MethodGet, skillsPath+"/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("reading skill: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading skill returned HTTP %d: %s", status, raw)
	}

	var s SkillResponse
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("reading skill: parsing response: %w", err)
	}
	return &s, nil
}

// Delete removes a skill and all its versions. 404 is treated as success.
// The API requires all versions to be deleted before the skill itself can be deleted.
func (c *SkillClient) Delete(ctx context.Context, id string) error {
	// List all versions and delete each one first.
	versions, err := c.listVersions(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting skill: listing versions: %w", err)
	}
	for _, v := range versions {
		raw, status, err := c.doJSON(ctx, http.MethodDelete,
			skillsPath+"/"+url.PathEscape(id)+"/versions/"+url.PathEscape(v)+"?beta=true", nil)
		if err != nil {
			return fmt.Errorf("deleting skill version %q: %w", v, err)
		}
		if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
			return fmt.Errorf("deleting skill version %q returned HTTP %d: %s", v, status, raw)
		}
	}

	_, status, err := c.doJSON(ctx, http.MethodDelete, skillsPath+"/"+url.PathEscape(id)+"?beta=true", nil)
	if err != nil {
		return fmt.Errorf("deleting skill: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("deleting skill returned HTTP %d", status)
	}
	return nil
}

// listVersions returns the numeric version timestamps for all versions of a skill.
func (c *SkillClient) listVersions(ctx context.Context, id string) ([]string, error) {
	raw, status, err := c.doJSON(ctx, http.MethodGet,
		skillsPath+"/"+url.PathEscape(id)+"/versions?beta=true", nil)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("listing versions returned HTTP %d: %s", status, raw)
	}
	var result struct {
		Data []struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parsing versions response: %w", err)
	}
	versions := make([]string, len(result.Data))
	for i, v := range result.Data {
		versions[i] = v.Version
	}
	return versions, nil
}

// CreateVersion uploads new source files as a new version of an existing skill.
func (c *SkillClient) CreateVersion(ctx context.Context, id, sourceDir string) (*SkillVersionResponse, error) {
	body, contentType, err := buildMultipartBody("", sourceDir)
	if err != nil {
		return nil, fmt.Errorf("creating skill version: building multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, auth.BaseURL+skillsPath+"/"+url.PathEscape(id)+"/versions", body)
	if err != nil {
		return nil, fmt.Errorf("creating skill version: building request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	if err := c.creds.Authenticate(ctx, req); err != nil {
		return nil, fmt.Errorf("creating skill version: authenticating request: %w", err)
	}

	raw, status, err := c.doRaw(req)
	if err != nil {
		return nil, fmt.Errorf("creating skill version: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating skill version returned HTTP %d: %s", status, raw)
	}

	var v SkillVersionResponse
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("creating skill version: parsing response: %w", err)
	}
	return &v, nil
}

// doJSON performs a JSON request using the standard doWithCreds helper.
func (c *SkillClient) doJSON(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	return doWithCreds(ctx, c.httpClient, c.creds, method, path, body)
}

// doRaw executes an already-built request and returns the response bytes and status code.
func (c *SkillClient) doRaw(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}
	return raw, resp.StatusCode, nil
}

// buildMultipartBody constructs the multipart/form-data body for skill create/version requests.
// displayTitle is only written when non-empty (Create passes it; CreateVersion passes "").
// It validates that sourceDir contains a SKILL.md at its root.
func buildMultipartBody(displayTitle, sourceDir string) (*bytes.Buffer, string, error) {
	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(sourceDir, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("walking source directory %q: %w", sourceDir, err)
	}
	if len(files) == 0 {
		return nil, "", fmt.Errorf("source directory %q is empty", sourceDir)
	}

	sort.Strings(files)

	skillMDFound := false
	for _, f := range files {
		if f == "SKILL.md" {
			skillMDFound = true
			break
		}
	}
	if !skillMDFound {
		return nil, "", fmt.Errorf("source directory %q is missing required SKILL.md at root", sourceDir)
	}

	// The API requires every file path to be prefixed with the skill name declared
	// in SKILL.md frontmatter (e.g. "my-skill/SKILL.md"). Parse the name now so we
	// can validate and prefix before opening the multipart writer.
	skillName, err := parseSkillName(filepath.Join(sourceDir, "SKILL.md"))
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if displayTitle != "" {
		if err := w.WriteField("display_title", displayTitle); err != nil {
			return nil, "", fmt.Errorf("writing display_title field: %w", err)
		}
	}

	for _, rel := range files {
		fw, err := w.CreateFormFile("files[]", skillName+"/"+rel)
		if err != nil {
			return nil, "", fmt.Errorf("creating form file for %q: %w", rel, err)
		}
		content, err := os.ReadFile(filepath.Join(sourceDir, rel))
		if err != nil {
			return nil, "", fmt.Errorf("reading file %q: %w", rel, err)
		}
		if _, err := fw.Write(content); err != nil {
			return nil, "", fmt.Errorf("writing file content for %q: %w", rel, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", err)
	}

	return &buf, w.FormDataContentType(), nil
}

// parseSkillName reads the YAML frontmatter from SKILL.md and returns the value
// of the `name` field. The API requires all file paths to be prefixed with this
// name (e.g. "my-skill/SKILL.md") and the prefix must match the name exactly.
func parseSkillName(skillMDPath string) (string, error) {
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return "", fmt.Errorf("reading SKILL.md: %w", err)
	}
	s := string(content)
	if !strings.HasPrefix(s, "---") {
		return "", fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}
	// Find the closing ---
	rest := s[3:]
	end := strings.Index(rest, "---")
	if end == -1 {
		return "", fmt.Errorf("SKILL.md frontmatter is not closed (missing second ---)")
	}
	frontmatter := rest[:end]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = strings.Trim(name, `"'`)
			if name == "" {
				return "", fmt.Errorf("SKILL.md frontmatter has empty 'name' field")
			}
			return name, nil
		}
	}
	return "", fmt.Errorf("SKILL.md frontmatter is missing required 'name' field")
}
