package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	anthropicBaseURL    = "https://api.anthropic.com"
	managedAgentsBeta   = "managed-agents-2026-04-01"
	anthropicAPIVersion = "2023-06-01"
)

func doRequest(ctx context.Context, data *providerData, method, path string, body any) ([]byte, int, error) {
	workspaceID, err := resolveWorkspaceID(ctx, data.apiKey, data.workspaceName)
	if err != nil {
		return nil, 0, fmt.Errorf("workspace resolution: %w", err)
	}

	token, err := mintToken(ctx, data.cfg, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("minting token: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, anthropicBaseURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("anthropic-beta", managedAgentsBeta)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}
