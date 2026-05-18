package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

func doWithCreds(ctx context.Context, httpClient *http.Client, creds auth.Credentials, method, path string, body any) ([]byte, int, error) {
	if httpClient == nil {
		return nil, 0, fmt.Errorf("http client is nil")
	}
	if creds == nil {
		return nil, 0, fmt.Errorf("credentials are nil")
	}
	req, err := buildRequest(ctx, method, auth.BaseURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	if err := creds.Authenticate(ctx, req); err != nil {
		return nil, 0, fmt.Errorf("authenticating request: %w", err)
	}

	resp, err := httpClient.Do(req)
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
	if body != nil {
		req.Header.Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
	}
	return req, nil
}
