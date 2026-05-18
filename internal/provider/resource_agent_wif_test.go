package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/auth"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// diagCollector collects AddError calls so tests can inspect them.
type diagCollector struct {
	diags diag.Diagnostics
}

func (d *diagCollector) AddError(summary, detail string) {
	d.diags.AddError(summary, detail)
}

func TestResolveCredentials_WIF(t *testing.T) {
	r := &WIFAgentResource{data: &providerData{
		wif: &auth.WIFConfig{},
	}}
	d := &diagCollector{}
	creds := r.resolveCredentials(context.Background(), "wrks_123", d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
	if _, ok := creds.(auth.WIFBearer); !ok {
		t.Errorf("expected WIFBearer, got %T", creds)
	}
}

func TestResolveCredentials_APIKey(t *testing.T) {
	r := &WIFAgentResource{data: &providerData{
		workspaceAPIKey: "sk-ant-api03-test",
	}}
	d := &diagCollector{}
	creds := r.resolveCredentials(context.Background(), "", d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
	if _, ok := creds.(auth.WorkspaceAPIKey); !ok {
		t.Errorf("expected WorkspaceAPIKey, got %T", creds)
	}
}

func TestResolveCredentials_WIFPrecedenceOverAPIKey(t *testing.T) {
	r := &WIFAgentResource{data: &providerData{
		wif:             &auth.WIFConfig{},
		workspaceAPIKey: "sk-ant-api03-test",
	}}
	d := &diagCollector{}
	creds := r.resolveCredentials(context.Background(), "wrks_123", d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
	if _, ok := creds.(auth.WIFBearer); !ok {
		t.Errorf("expected WIFBearer (WIF takes precedence), got %T", creds)
	}
}

func TestResolveCredentials_NeitherConfigured(t *testing.T) {
	r := &WIFAgentResource{data: &providerData{}}
	d := &diagCollector{}
	creds := r.resolveCredentials(context.Background(), "", d)
	if !d.diags.HasError() {
		t.Fatal("expected error when no credentials configured")
	}
	if creds != nil {
		t.Error("expected nil credentials on error")
	}
}

func TestResolveCredentials_WIFErrorSurfaced(t *testing.T) {
	r := &WIFAgentResource{data: &providerData{
		wifErr: fmt.Errorf("incomplete WIF configuration, missing: federation_rule_id"),
	}}
	d := &diagCollector{}
	r.resolveCredentials(context.Background(), "wrks_123", d)
	if !d.diags.HasError() {
		t.Fatal("expected error when workspace_id set and WIF has error")
	}
}

func TestResolveCredentials_NilProviderData(t *testing.T) {
	r := &WIFAgentResource{}
	d := &diagCollector{}
	creds := r.resolveCredentials(context.Background(), "", d)
	if !d.diags.HasError() {
		t.Fatal("expected error when provider data is nil")
	}
	if creds != nil {
		t.Error("expected nil credentials")
	}
}

func TestWorkspaceAPIKey_Authenticate(t *testing.T) {
	key := auth.WorkspaceAPIKey{Key: "sk-ant-api03-test"}
	req, _ := http.NewRequest(http.MethodGet, "https://api.anthropic.com/v1/agents", nil)
	if err := key.Authenticate(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("x-api-key"); got != "sk-ant-api03-test" {
		t.Errorf("expected x-api-key header, got %q", got)
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
