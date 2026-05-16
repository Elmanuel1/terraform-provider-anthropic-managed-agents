package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
)

type Config struct {
	WIF        *auth.WIFConfig
	APIKey     string // admin API key, used by workspace resource and token datasource
	HTTPClient *http.Client
}

func (c *Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func doWithCreds(ctx context.Context, cfg *Config, creds auth.Credentials, method, path string, body any) ([]byte, int, error) {
	req, err := buildRequest(ctx, method, auth.BaseURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	if err := creds.Authenticate(ctx, req); err != nil {
		return nil, 0, fmt.Errorf("authenticating request: %w", err)
	}

	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}

func buildRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
	return req, nil
}
