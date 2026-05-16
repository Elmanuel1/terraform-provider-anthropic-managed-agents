package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
)

const workspacesPath = "/v1/organizations/workspaces"

type WorkspaceResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"created_at"`
	ArchivedAt *string `json:"archived_at"`
}

type WorkspaceClient struct {
	apiKey     string
	httpClient *http.Client
	once       sync.Once
	byName     map[string]string
	fetchErr   error
}

func NewWorkspaceClient(apiKey string, httpClient *http.Client) *WorkspaceClient {
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}
	return &WorkspaceClient{apiKey: apiKey, httpClient: httpClient}
}

func (c *WorkspaceClient) creds() auth.Credentials {
	return auth.AdminAPIKey{Key: c.apiKey}
}

// ResolveByName resolves a workspace name to its ID via the Admin API.
// The workspace list is fetched once per client instance and cached.
func (c *WorkspaceClient) ResolveByName(ctx context.Context, name string) (string, error) {
	c.once.Do(func() {
		raw, status, err := doWithCreds(ctx, c.httpClient, c.creds(), http.MethodGet, workspacesPath, nil)
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
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			c.fetchErr = fmt.Errorf("parsing workspaces response: %w", err)
			return
		}

		c.byName = make(map[string]string, len(result.Data))
		for _, w := range result.Data {
			c.byName[w.Name] = w.ID
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
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds(), http.MethodPost, workspacesPath, body)
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
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds(), http.MethodGet, workspacesPath+"/"+id, nil)
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

func (c *WorkspaceClient) Delete(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds(), http.MethodDelete, workspacesPath+"/"+id, nil)
	if err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("deleting workspace returned HTTP %d", status)
	}
	return nil
}
