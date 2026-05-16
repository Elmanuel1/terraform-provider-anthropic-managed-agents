package client

import (
	"net/http"
	"testing"
	"time"
)

func TestHTTPClient_DefaultTimeout(t *testing.T) {
	cfg := &Config{}
	c := cfg.httpClient()
	if c.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.Timeout)
	}
}

func TestHTTPClient_Custom(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	cfg := &Config{HTTPClient: custom}
	if cfg.httpClient() != custom {
		t.Error("expected the injected client to be returned")
	}
}
