package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
)

type AgentResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model struct {
		ID    string `json:"id"`
		Speed string `json:"speed"`
	} `json:"model"`
	System      *string           `json:"system"`
	Description *string           `json:"description"`
	Tools       []json.RawMessage `json:"tools"`
	MCPServers  []json.RawMessage `json:"mcp_servers"`
	Skills      []json.RawMessage `json:"skills"`
	Multiagent  *json.RawMessage  `json:"multiagent"`
	Metadata    map[string]string `json:"metadata"`
	Version     int               `json:"version"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	ArchivedAt  *string           `json:"archived_at"`
}

type AgentClient struct {
	creds      auth.Credentials
	httpClient *http.Client
}

func NewAgentClient(creds auth.WIFBearer) *AgentClient {
	return &AgentClient{creds: creds, httpClient: defaultHTTPClient}
}

func (c *AgentClient) Create(ctx context.Context, body map[string]any) (*AgentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/agents", body)
	if err != nil {
		return nil, fmt.Errorf("creating agent: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("creating agent returned HTTP %d: %s", status, raw)
	}

	var a AgentResponse
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("parsing agent response: %w", err)
	}
	return &a, nil
}

func (c *AgentClient) Read(ctx context.Context, id string) (*AgentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodGet, "/v1/agents/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("reading agent: %w", err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("reading agent returned HTTP %d: %s", status, raw)
	}

	var a AgentResponse
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("parsing agent response: %w", err)
	}
	return &a, nil
}

func (c *AgentClient) Update(ctx context.Context, id string, body map[string]any) (*AgentResponse, error) {
	raw, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/agents/"+url.PathEscape(id), body)
	if err != nil {
		return nil, fmt.Errorf("updating agent: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("updating agent returned HTTP %d: %s", status, raw)
	}

	var a AgentResponse
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("parsing agent response: %w", err)
	}
	return &a, nil
}

// Delete archives the agent via POST /v1/agents/{id}/archive.
func (c *AgentClient) Delete(ctx context.Context, id string) error {
	_, status, err := doWithCreds(ctx, c.httpClient, c.creds, http.MethodPost, "/v1/agents/"+url.PathEscape(id)+"/archive", nil)
	if err != nil {
		return fmt.Errorf("archiving agent: %w", err)
	}
	if status != http.StatusOK && status != http.StatusNotFound {
		return fmt.Errorf("archiving agent returned HTTP %d", status)
	}
	return nil
}
