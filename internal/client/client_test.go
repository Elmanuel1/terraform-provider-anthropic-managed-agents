package client

import (
	"net/http"
	"testing"
	"time"
)

func TestNewWorkspaceClient_DefaultHTTPClient(t *testing.T) {
	c := NewWorkspaceClient("key", nil)
	if c.httpClient != defaultHTTPClient {
		t.Error("expected defaultHTTPClient when nil is passed")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.httpClient.Timeout)
	}
}

func TestNewWorkspaceClient_CustomHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c := NewWorkspaceClient("key", custom)
	if c.httpClient != custom {
		t.Error("expected injected client to be used")
	}
}
