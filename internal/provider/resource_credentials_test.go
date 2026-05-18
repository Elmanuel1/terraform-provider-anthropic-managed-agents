package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// diagCollector collects AddError calls so tests can inspect them.
type diagCollector struct {
	diags diag.Diagnostics
}

func (d *diagCollector) AddError(summary, detail string) {
	d.diags.AddError(summary, detail)
}

// resolveWorkspaceCredentials

func TestResolveWorkspaceCredentials_WIF(t *testing.T) {
	d := &diagCollector{}
	creds := resolveWorkspaceCredentials(context.Background(), &providerData{wif: &auth.WIFConfig{}}, "anthropic_vault", "wrks_123", d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
	if _, ok := creds.(auth.WIFBearer); !ok {
		t.Errorf("expected WIFBearer, got %T", creds)
	}
}

func TestResolveWorkspaceCredentials_APIKey(t *testing.T) {
	d := &diagCollector{}
	creds := resolveWorkspaceCredentials(context.Background(), &providerData{workspaceAPIKey: "sk-ant-api03-test"}, "anthropic_vault", "", d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
	if _, ok := creds.(auth.WorkspaceAPIKey); !ok {
		t.Errorf("expected WorkspaceAPIKey, got %T", creds)
	}
}

func TestResolveWorkspaceCredentials_WIFPrecedenceOverAPIKey(t *testing.T) {
	d := &diagCollector{}
	creds := resolveWorkspaceCredentials(context.Background(), &providerData{
		wif:             &auth.WIFConfig{},
		workspaceAPIKey: "sk-ant-api03-test",
	}, "anthropic_vault", "wrks_123", d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
	if _, ok := creds.(auth.WIFBearer); !ok {
		t.Errorf("expected WIFBearer (WIF takes precedence), got %T", creds)
	}
}

func TestResolveWorkspaceCredentials_WIFNoWorkspaceID_FallsBackToAPIKey(t *testing.T) {
	// WIF is configured but workspace_id is empty — should fall back to API key.
	d := &diagCollector{}
	creds := resolveWorkspaceCredentials(context.Background(), &providerData{
		wif:             &auth.WIFConfig{},
		workspaceAPIKey: "sk-ant-api03-test",
	}, "anthropic_vault", "", d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
	if _, ok := creds.(auth.WorkspaceAPIKey); !ok {
		t.Errorf("expected WorkspaceAPIKey fallback, got %T", creds)
	}
}

func TestResolveWorkspaceCredentials_NeitherConfigured(t *testing.T) {
	d := &diagCollector{}
	creds := resolveWorkspaceCredentials(context.Background(), &providerData{}, "anthropic_vault", "", d)
	if !d.diags.HasError() {
		t.Fatal("expected error when no credentials configured")
	}
	if creds != nil {
		t.Error("expected nil credentials on error")
	}
}

func TestResolveWorkspaceCredentials_WIFErrorSurfaced(t *testing.T) {
	d := &diagCollector{}
	resolveWorkspaceCredentials(context.Background(), &providerData{
		wifErr: fmt.Errorf("missing federation_rule_id"),
	}, "anthropic_vault", "wrks_123", d)
	if !d.diags.HasError() {
		t.Fatal("expected error when workspace_id set and WIF has error")
	}
}

func TestResolveWorkspaceCredentials_NilProviderData(t *testing.T) {
	d := &diagCollector{}
	creds := resolveWorkspaceCredentials(context.Background(), nil, "anthropic_vault", "", d)
	if !d.diags.HasError() {
		t.Fatal("expected error when provider data is nil")
	}
	if creds != nil {
		t.Error("expected nil credentials")
	}
}

// validateWorkspaceCredentials

func TestValidateWorkspaceCredentials_NeitherConfigured(t *testing.T) {
	d := &diagCollector{}
	validateWorkspaceCredentials(&providerData{}, "anthropic_vault", "", false, d)
	if !d.diags.HasError() {
		t.Fatal("expected error when no credentials configured")
	}
}

func TestValidateWorkspaceCredentials_WIFOnlyMissingWorkspaceID(t *testing.T) {
	d := &diagCollector{}
	validateWorkspaceCredentials(&providerData{wif: &auth.WIFConfig{}}, "anthropic_vault", "", false, d)
	if !d.diags.HasError() {
		t.Fatal("expected error when WIF configured without workspace_id")
	}
}

func TestValidateWorkspaceCredentials_WIFOnlyWorkspaceIDUnknown_NoError(t *testing.T) {
	// workspace_id references a not-yet-created resource — should not error at plan time.
	d := &diagCollector{}
	validateWorkspaceCredentials(&providerData{wif: &auth.WIFConfig{}}, "anthropic_vault", "", true, d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error when workspace_id is unknown: %s", d.diags[0].Detail())
	}
}

func TestValidateWorkspaceCredentials_WIFWithWorkspaceID_NoError(t *testing.T) {
	d := &diagCollector{}
	validateWorkspaceCredentials(&providerData{wif: &auth.WIFConfig{}}, "anthropic_vault", "wrks_123", false, d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
}

func TestValidateWorkspaceCredentials_APIKeyOnly_NoError(t *testing.T) {
	d := &diagCollector{}
	validateWorkspaceCredentials(&providerData{workspaceAPIKey: "sk-ant-api03-test"}, "anthropic_vault", "", false, d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
}

func TestValidateWorkspaceCredentials_NilData_NoError(t *testing.T) {
	// nil providerData means Configure hasn't run yet — skip silently.
	d := &diagCollector{}
	validateWorkspaceCredentials(nil, "anthropic_vault", "", false, d)
	if d.diags.HasError() {
		t.Fatalf("unexpected error: %s", d.diags[0].Detail())
	}
}
