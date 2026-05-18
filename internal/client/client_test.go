package client

import (
	"testing"
	"time"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

func TestDefaultHTTPClient_Timeout(t *testing.T) {
	c := NewWorkspaceClient(auth.AdminAPIKey{Key: "key"})
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.httpClient.Timeout)
	}
	if c.httpClient != defaultHTTPClient {
		t.Error("expected shared defaultHTTPClient")
	}
}
