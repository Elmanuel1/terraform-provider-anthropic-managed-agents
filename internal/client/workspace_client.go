package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

const workspacesPath = "/v1/organizations/workspaces"

type WorkspaceResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"created_at"`
	ArchivedAt *string `json:"archived_at"`
}

type WorkspaceClient struct {
	creds      auth.Credentials
	httpClient *http.Client
	once       sync.Once
	byName     map[string]string
	fetchErr   error
}

func NewWorkspaceClient(creds auth.Credentials) *WorkspaceClient {
	return &WorkspaceClient{creds: creds, httpClient: defaultHTTPClient}
}

// ResolveByName resolves a workspace name to its ID via the Admin API.
// The full workspace list is fetched (with pagination) once per client instance and cached.
// Note: sync.Once captures the context from the first caller. If that context
// is cancelled during the fetch, fetchErr is set permanently and all subsequent
// calls return it. Terraform provider contexts are long-lived so this is safe in practice.
func (c *WorkspaceClient) ResolveByName(ctx context.Context, name string) (string, error) {
	c.once.Do(func() {
		c.byName = make(map[string]string)
		afterID := ""
		for {
			path := workspacesPath
			if afterID != "" {
				path += "?after_id=" + url.QueryEscape(afterID)
			}
			raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, path, nil)
			if err != nil {
				c.fetchErr = fmt.Errorf("listing workspaces: %w", err)
				return
			}
			if status != http.StatusOK {
				c.fetchErr = fmt.Errorf("listing workspaces returned HTTP %d: %s", status, raw)
				return
			}

			var result struct {
				Data []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"data"`
				HasMore bool   `json:"has_more"`
				LastID  string `json:"last_id"`
			}
			if err := json.Unmarshal(raw, &result); err != nil {
				c.fetchErr = fmt.Errorf("parsing workspaces response: %w", err)
				return
			}

			for _, w := range result.Data {
				c.byName[w.Name] = w.ID
			}

			if !result.HasMore || result.LastID == "" {
				break
			}
			afterID = result.LastID
		}
	})

	if c.fetchErr != nil {
		return "", c.fetchErr
	}

	if id, ok := c.byName[name]; ok {
		return id, nil
	}

	available := make([]string, 0, len(c.byName))
	for n := range c.byName {
		available = append(available, fmt.Sprintf("%q", n))
	}
	return "", fmt.Errorf("workspace %q not found — available: [%s]", name, strings.Join(available, ", "))
}

func (c *WorkspaceClient) Create(ctx context.Context, name string) (*WorkspaceResponse, error) {
	body := map[string]any{"name": name}
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, workspacesPath, body)
	if err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating workspace returned HTTP %d: %s", status, raw)
	}

	var w WorkspaceResponse
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("parsing workspace response: %w", err)
	}
	return &w, nil
}

func (c *WorkspaceClient) Read(ctx context.Context, id string) (*WorkspaceResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, workspacesPath+"/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("reading workspace: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading workspace returned HTTP %d: %s", status, raw)
	}

	var w WorkspaceResponse
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("parsing workspace response: %w", err)
	}
	return &w, nil
}

func (c *WorkspaceClient) Update(ctx context.Context, id string, body map[string]any) (*WorkspaceResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, workspacesPath+"/"+url.PathEscape(id), body)
	if err != nil {
		return nil, fmt.Errorf("updating workspace: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("updating workspace returned HTTP %d: %s", status, raw)
	}
	var w WorkspaceResponse
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("parsing workspace response: %w", err)
	}
	return &w, nil
}

func (c *WorkspaceClient) Archive(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, workspacesPath+"/"+url.PathEscape(id)+"/archive", nil)
	if err != nil {
		return fmt.Errorf("archiving workspace: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("archiving workspace returned HTTP %d", status)
	}
	return nil
}
