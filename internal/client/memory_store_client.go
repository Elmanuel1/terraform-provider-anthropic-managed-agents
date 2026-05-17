package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
)

type MemoryStoreResponse struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	ArchivedAt  *string           `json:"archived_at"`
}

type MemoryStoreClient struct {
	creds      auth.Credentials
	httpClient *http.Client
}

func NewMemoryStoreClient(creds auth.WIFBearer) *MemoryStoreClient {
	return &MemoryStoreClient{creds: creds, httpClient: defaultHTTPClient}
}

func (c *MemoryStoreClient) Create(ctx context.Context, body map[string]any) (*MemoryStoreResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/memory_stores", body)
	if err != nil {
		return nil, fmt.Errorf("creating memory store: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating memory store returned HTTP %d: %s", status, raw)
	}

	var s MemoryStoreResponse
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parsing memory store response: %w", err)
	}
	return &s, nil
}

func (c *MemoryStoreClient) Read(ctx context.Context, id string) (*MemoryStoreResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, "/v1/memory_stores/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("reading memory store: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading memory store returned HTTP %d: %s", status, raw)
	}

	var s MemoryStoreResponse
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parsing memory store response: %w", err)
	}
	return &s, nil
}

func (c *MemoryStoreClient) Update(ctx context.Context, id string, body map[string]any) (*MemoryStoreResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/memory_stores/"+url.PathEscape(id), body)
	if err != nil {
		return nil, fmt.Errorf("updating memory store: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("updating memory store returned HTTP %d: %s", status, raw)
	}

	var s MemoryStoreResponse
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parsing memory store response: %w", err)
	}
	return &s, nil
}

func (c *MemoryStoreClient) Archive(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/memory_stores/"+url.PathEscape(id)+"/archive", nil)
	if err != nil {
		return fmt.Errorf("archiving memory store: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("archiving memory store returned HTTP %d", status)
	}
	return nil
}

func (c *MemoryStoreClient) Delete(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodDelete, "/v1/memory_stores/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("deleting memory store: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		return fmt.Errorf("deleting memory store returned HTTP %d", status)
	}
	return nil
}
