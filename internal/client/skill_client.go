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

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

const skillsPath = "/v1/skills"

type SkillResponse struct {
	ID           string  `json:"id"`
	DisplayTitle string  `json:"display_title"`
	Description  *string `json:"description"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type SkillVersionResponse struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

type SkillClient struct {
	creds      auth.AdminAPIKey
	httpClient *http.Client
}

func NewSkillClient(creds auth.AdminAPIKey) *SkillClient {
	return &SkillClient{creds: creds, httpClient: defaultHTTPClient}
}

// Create uploads a new skill from sourceDir via multipart POST to /v1/skills?beta=true.
// display_title is required; description is optional.
func (c *SkillClient) Create(ctx context.Context, displayTitle string, description *string, sourceDir string) (*SkillResponse, error) {
	body, contentType, err := buildMultipartBody(displayTitle, description, sourceDir)
	if err != nil {
		return nil, fmt.Errorf("creating skill: building multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, auth.BaseURL+skillsPath+"?beta=true", body)
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

	// The create endpoint may return a minimal response; do a Read to get full state.
	var partial struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &partial); err != nil {
		return nil, fmt.Errorf("creating skill: parsing response: %w", err)
	}
	if partial.ID == "" {
		return nil, fmt.Errorf("creating skill: response did not include an id: %s", raw)
	}

	return c.Read(ctx, partial.ID)
}

// Read fetches a skill by ID from GET /v1/skills/{id}?beta=true.
// Returns nil if the skill is not found (404).
func (c *SkillClient) Read(ctx context.Context, id string) (*SkillResponse, error) {
	raw, status, err := c.doJSON(ctx, http.MethodGet, skillsPath+"/"+url.PathEscape(id)+"?beta=true", nil)
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

// Delete removes a skill. 404 is treated as success (already deleted).
func (c *SkillClient) Delete(ctx context.Context, id string) error {
	_, status, err := c.doJSON(ctx, http.MethodDelete, skillsPath+"/"+url.PathEscape(id)+"?beta=true", nil)
	if err != nil {
		return fmt.Errorf("deleting skill: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("deleting skill returned HTTP %d", status)
	}
	return nil
}

// CreateVersion uploads new source files as a new version of an existing skill.
func (c *SkillClient) CreateVersion(ctx context.Context, id, sourceDir string) (*SkillVersionResponse, error) {
	body, contentType, err := buildMultipartBody("", nil, sourceDir)
	if err != nil {
		return nil, fmt.Errorf("creating skill version: building multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, auth.BaseURL+skillsPath+"/"+url.PathEscape(id)+"/versions?beta=true", body)
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
// displayTitle and description are only included when non-empty (for Create, not CreateVersion).
// It validates that sourceDir contains a SKILL.md at its root.
func buildMultipartBody(displayTitle string, description *string, sourceDir string) (*bytes.Buffer, string, error) {
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

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if displayTitle != "" {
		if err := w.WriteField("display_title", displayTitle); err != nil {
			return nil, "", fmt.Errorf("writing display_title field: %w", err)
		}
	}

	if description != nil && *description != "" {
		if err := w.WriteField("description", *description); err != nil {
			return nil, "", fmt.Errorf("writing description field: %w", err)
		}
	}

	for _, rel := range files {
		fw, err := w.CreateFormFile("files", rel)
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
