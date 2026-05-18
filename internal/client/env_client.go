package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

type EnvironmentResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Config      *struct {
		Packages   json.RawMessage `json:"packages"`
		Networking *struct {
			Type                 string   `json:"type"`
			AllowedHosts         []string `json:"allowed_hosts"`
			AllowMCPServers      *bool    `json:"allow_mcp_servers"`
			AllowPackageManagers *bool    `json:"allow_package_managers"`
		} `json:"networking"`
	} `json:"config"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
	ArchivedAt *string           `json:"archived_at"`
}

type EnvironmentClient struct {
	creds      auth.Credentials
	httpClient *http.Client
}

func NewEnvironmentClient(creds auth.WIFBearer) *EnvironmentClient {
	return &EnvironmentClient{creds: creds, httpClient: defaultHTTPClient}
}

func (c *EnvironmentClient) Create(ctx context.Context, body map[string]any) (*EnvironmentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/environments", body)
	if err != nil {
		return nil, fmt.Errorf("creating environment: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating environment returned HTTP %d: %s", status, raw)
	}

	var e EnvironmentResponse
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("parsing environment response: %w", err)
	}
	return &e, nil
}

func (c *EnvironmentClient) Read(ctx context.Context, id string) (*EnvironmentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, "/v1/environments/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("reading environment: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading environment returned HTTP %d: %s", status, raw)
	}

	var e EnvironmentResponse
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("parsing environment response: %w", err)
	}
	return &e, nil
}

func (c *EnvironmentClient) Update(ctx context.Context, id string, body map[string]any) (*EnvironmentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/environments/"+url.PathEscape(id), body)
	if err != nil {
		return nil, fmt.Errorf("updating environment: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("updating environment returned HTTP %d: %s", status, raw)
	}

	var e EnvironmentResponse
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("parsing environment response: %w", err)
	}
	return &e, nil
}

func (c *EnvironmentClient) Archive(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/environments/"+url.PathEscape(id)+"/archive", nil)
	if err != nil {
		return fmt.Errorf("archiving environment: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("archiving environment returned HTTP %d", status)
	}
	return nil
}

func (c *EnvironmentClient) Delete(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodDelete, "/v1/environments/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("deleting environment: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("deleting environment returned HTTP %d", status)
	}
	return nil
}
