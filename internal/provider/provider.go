package provider

import (
	"context"
	"os"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func New() provider.Provider {
	return &wifProvider{}
}

type wifProvider struct{}

type providerData struct {
	apiKey string
	wif    *auth.WIFConfig
}

func (p *wifProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anthropic-wif"
}

func (p *wifProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Anthropic provider with WIF token minting via TFC OIDC. All config via environment variables.\n\n" +
			"ANTHROPIC_ADMIN_API_KEY is always required.\n" +
			"WIF env vars (ANTHROPIC_FEDERATION_RULE_ID, ANTHROPIC_ORGANIZATION_ID, ANTHROPIC_SERVICE_ACCOUNT_ID, TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC) " +
			"are required only when managing agents or environments.",
	}
}

func (p *wifProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	apiKey := os.Getenv("ANTHROPIC_ADMIN_API_KEY")
	if apiKey == "" {
		resp.Diagnostics.AddError("Missing ANTHROPIC_ADMIN_API_KEY", "Required for all Admin API operations.")
		return
	}

	// WIF is optional at configure time — required only when agent/environment resources are used.
	wifCfg, err := auth.ReadWIFConfig()
	if err != nil {
		resp.Diagnostics.AddError("WIF configuration error", err.Error())
		return
	}

	data := &providerData{
		apiKey: apiKey,
		wif:    wifCfg,
	}
	resp.DataSourceData = data
	resp.ResourceData = data

	if wifCfg != nil {
		tflog.Info(ctx, "provider configured", map[string]any{
			"federation_rule_id":  wifCfg.FederationRuleID,
			"service_account_id": wifCfg.ServiceAccountID,
		})
	} else {
		tflog.Info(ctx, "provider configured — workspace-only mode (no WIF)")
	}
}

func (p *wifProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *wifProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource,
		NewAgentResource,
		NewEnvironmentResource,
	}
}
