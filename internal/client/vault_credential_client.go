package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

type VaultCredentialAuthResponse struct {
	Type         string                          `json:"type"`
	MCPServerURL *string                         `json:"mcp_server_url,omitempty"`
	ExpiresAt    *string                         `json:"expires_at,omitempty"`
	Refresh      *VaultCredentialRefreshResponse `json:"refresh,omitempty"`
}

type VaultCredentialRefreshResponse struct {
	ClientID          string  `json:"client_id,omitempty"`
	TokenEndpoint     string  `json:"token_endpoint,omitempty"`
	TokenEndpointAuth *struct {
		Type string `json:"type"`
	} `json:"token_endpoint_auth,omitempty"`
	Resource *string `json:"resource,omitempty"`
	Scope    *string `json:"scope,omitempty"`
}

type VaultCredentialResponse struct {
	ID          string                      `json:"id"`
	Type        string                      `json:"type"`
	VaultID     string                      `json:"vault_id"`
	DisplayName *string                     `json:"display_name"`
	Auth        VaultCredentialAuthResponse `json:"auth"`
	Metadata    map[string]string           `json:"metadata"`
	CreatedAt   string                      `json:"created_at"`
	UpdatedAt   string                      `json:"updated_at"`
	ArchivedAt  *string                     `json:"archived_at"`
}

type VaultCredentialClient struct {
	creds      auth.Credentials
	httpClient *http.Client
}

func NewVaultCredentialClient(creds auth.WIFBearer) *VaultCredentialClient {
	return &VaultCredentialClient{creds: creds, httpClient: defaultHTTPClient}
}

func (c *VaultCredentialClient) Create(ctx context.Context, vaultID string, body map[string]any) (*VaultCredentialResponse, error) {
	path := "/v1/vaults/" + url.PathEscape(vaultID) + "/credentials"
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("creating vault credential: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating vault credential returned HTTP %d: %s", status, raw)
	}

	var cr VaultCredentialResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("parsing vault credential response: %w", err)
	}
	return &cr, nil
}

func (c *VaultCredentialClient) Read(ctx context.Context, vaultID, id string) (*VaultCredentialResponse, error) {
	path := "/v1/vaults/" + url.PathEscape(vaultID) + "/credentials/" + url.PathEscape(id)
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("reading vault credential: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading vault credential returned HTTP %d: %s", status, raw)
	}

	var cr VaultCredentialResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("parsing vault credential response: %w", err)
	}
	return &cr, nil
}

func (c *VaultCredentialClient) Update(ctx context.Context, vaultID, id string, body map[string]any) (*VaultCredentialResponse, error) {
	path := "/v1/vaults/" + url.PathEscape(vaultID) + "/credentials/" + url.PathEscape(id)
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("updating vault credential: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("updating vault credential returned HTTP %d: %s", status, raw)
	}

	var cr VaultCredentialResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("parsing vault credential response: %w", err)
	}
	return &cr, nil
}

func (c *VaultCredentialClient) Archive(ctx context.Context, vaultID, id string) error {
	path := "/v1/vaults/" + url.PathEscape(vaultID) + "/credentials/" + url.PathEscape(id) + "/archive"
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("archiving vault credential: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("archiving vault credential returned HTTP %d", status)
	}
	return nil
}

func (c *VaultCredentialClient) Delete(ctx context.Context, vaultID, id string) error {
	path := "/v1/vaults/" + url.PathEscape(vaultID) + "/credentials/" + url.PathEscape(id)
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting vault credential: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("deleting vault credential returned HTTP %d", status)
	}
	return nil
}
