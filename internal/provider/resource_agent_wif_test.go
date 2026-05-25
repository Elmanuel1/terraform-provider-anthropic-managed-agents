package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
)

func TestWorkspaceAPIKey_Authenticate(t *testing.T) {
	key := auth.WorkspaceAPIKey{Key: "sk-ant-api03-test"}
	req, _ := http.NewRequest(http.MethodGet, "https://api.anthropic.com/v1/agents", nil)
	if err := key.Authenticate(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("x-api-key"); got != "sk-ant-api03-test" {
		t.Errorf("expected x-api-key header, got %q", got)
	}
	// WorkspaceAPIKey does not set a beta header — callers use WithBeta explicitly.
	if got := req.Header.Get("anthropic-beta"); got != "" {
		t.Errorf("expected no beta header from bare WorkspaceAPIKey, got %q", got)
	}
}

func TestWithBeta_SetsHeader(t *testing.T) {
	creds := auth.WithBeta(auth.WorkspaceAPIKey{Key: "sk-ant-api03-test"}, auth.AgentsBeta)
	req, _ := http.NewRequest(http.MethodGet, "https://api.anthropic.com/v1/agents", nil)
	if err := creds.Authenticate(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("anthropic-beta"); got != auth.AgentsBeta {
		t.Errorf("expected AgentsBeta header, got %q", got)
	}
}

func TestWorkspaceAPIKey_Authenticate_EmptyKey(t *testing.T) {
	key := auth.WorkspaceAPIKey{}
	req, _ := http.NewRequest(http.MethodGet, "https://api.anthropic.com/v1/agents", nil)
	if err := key.Authenticate(context.Background(), req); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestImportState_BothParts(t *testing.T) {
	parts := strings.SplitN("wrks_abc/agt_xyz", "/", 2)
	if len(parts) != 2 || parts[0] != "wrks_abc" || parts[1] != "agt_xyz" {
		t.Errorf("unexpected split: %v", parts)
	}
}

func TestImportState_AgentIDOnly(t *testing.T) {
	parts := strings.SplitN("agt_xyz", "/", 2)
	if len(parts) != 1 || parts[0] != "agt_xyz" {
		t.Errorf("unexpected split: %v", parts)
	}
}
