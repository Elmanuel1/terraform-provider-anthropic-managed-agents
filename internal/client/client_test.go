package client

import (
	"testing"
	"time"
)

func TestDefaultHTTPClient_Timeout(t *testing.T) {
	c := NewWorkspaceClient("key")
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.httpClient.Timeout)
	}
	if c.httpClient != defaultHTTPClient {
		t.Error("expected shared defaultHTTPClient")
	}
}
