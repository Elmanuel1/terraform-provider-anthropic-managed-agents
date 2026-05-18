package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/auth"
)

type VaultResponse struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	DisplayName string            `json:"display_name"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	ArchivedAt  *string           `json:"archived_at"`
}

type VaultClient struct {
	creds      auth.Credentials
	httpClient *http.Client
}

func NewVaultClient(creds auth.WIFBearer) *VaultClient {
	return &VaultClient{creds: creds, httpClient: defaultHTTPClient}
}

func (c *VaultClient) Create(ctx context.Context, body map[string]any) (*VaultResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/vaults", body)
	if err != nil {
		return nil, fmt.Errorf("creating vault: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating vault returned HTTP %d: %s", status, raw)
	}

	var v VaultResponse
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("parsing vault response: %w", err)
	}
	return &v, nil
}

func (c *VaultClient) Read(ctx context.Context, id string) (*VaultResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, "/v1/vaults/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("reading vault: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading vault returned HTTP %d: %s", status, raw)
	}

	var v VaultResponse
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("parsing vault response: %w", err)
	}
	return &v, nil
}

func (c *VaultClient) Update(ctx context.Context, id string, body map[string]any) (*VaultResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/vaults/"+url.PathEscape(id), body)
	if err != nil {
		return nil, fmt.Errorf("updating vault: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("updating vault returned HTTP %d: %s", status, raw)
	}

	var v VaultResponse
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("parsing vault response: %w", err)
	}
	return &v, nil
}

func (c *VaultClient) Archive(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/vaults/"+url.PathEscape(id)+"/archive", nil)
	if err != nil {
		return fmt.Errorf("archiving vault: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("archiving vault returned HTTP %d", status)
	}
	return nil
}

func (c *VaultClient) Delete(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodDelete, "/v1/vaults/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("deleting vault: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("deleting vault returned HTTP %d", status)
	}
	return nil
}

