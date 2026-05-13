package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func New() provider.Provider {
	return &wifProvider{}
}

type wifProvider struct {
	cfg *wifConfig
}

type providerData struct {
	cfg    *wifConfig
	apiKey string
}

func (p *wifProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anthropic-wif"
}

func (p *wifProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Validates Anthropic WIF token minting via TFC OIDC. No attributes needed — all config via environment variables.",
	}
}

func (p *wifProvider) Configure(ctx context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	cfg, err := readWIFConfig()
	if err != nil {
		resp.Diagnostics.AddError("WIF configuration error", err.Error())
		return
	}
	if cfg == nil {
		resp.Diagnostics.AddError(
			"WIF not configured",
			"Set ANTHROPIC_FEDERATION_RULE_ID, ANTHROPIC_ORGANIZATION_ID, ANTHROPIC_SERVICE_ACCOUNT_ID, and TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC.",
		)
		return
	}

	apiKey := os.Getenv("ANTHROPIC_ADMIN_API_KEY")
	if apiKey == "" {
		resp.Diagnostics.AddError("Missing ANTHROPIC_ADMIN_API_KEY", "Required for workspace name resolution via Admin API.")
		return
	}

	data := &providerData{cfg: cfg, apiKey: apiKey}
	resp.DataSourceData = data
	resp.ResourceData = data

	fmt.Printf("[anthropic-wif] provider configured — federation_rule_id=%s service_account_id=%s\n",
		cfg.FederationRuleID, cfg.ServiceAccountID)
}

func (p *wifProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &tokenDataSource{} },
	}
}

func (p *wifProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
