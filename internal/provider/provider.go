package provider

import (
	"context"
	"os"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/auth"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func New() provider.Provider {
	return &anthropicProvider{}
}

type anthropicProvider struct{}

type providerConfig struct {
	AdminAPIKey      types.String `tfsdk:"admin_api_key"`
	FederationRuleID types.String `tfsdk:"federation_rule_id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
}

type providerData struct {
	adminKey string
	wif      *auth.WIFConfig
	wifErr   error
}

func (p *anthropicProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anthropic"
}

func (p *anthropicProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Anthropic platform resources. " +
			"All attributes are optional in the provider block and fall back to environment variables. " +
			"Each resource validates only the credentials it needs at operation time.",
		Attributes: map[string]schema.Attribute{
			"admin_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Anthropic Admin API key (sk-ant-admin-...). Falls back to ANTHROPIC_ADMIN_API_KEY. Required for anthropic_workspace and anthropic_memory_store.",
			},
			"federation_rule_id": schema.StringAttribute{
				Optional:    true,
				Description: "Federation rule ID (fdrl_...). Falls back to ANTHROPIC_FEDERATION_RULE_ID. Required for anthropic_wif_* resources.",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Description: "Anthropic organization UUID. Falls back to ANTHROPIC_ORGANIZATION_ID. Required for anthropic_wif_* resources.",
			},
			"service_account_id": schema.StringAttribute{
				Optional:    true,
				Description: "Service account ID (svac_...). Falls back to ANTHROPIC_SERVICE_ACCOUNT_ID. Required for anthropic_wif_* resources.",
			},
		},
	}
}

func (p *anthropicProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerConfig
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	adminKey := firstNonEmpty(cfg.AdminAPIKey.ValueString(), os.Getenv("ANTHROPIC_ADMIN_API_KEY"))
	ruleID := firstNonEmpty(cfg.FederationRuleID.ValueString(), os.Getenv("ANTHROPIC_FEDERATION_RULE_ID"))
	orgID := firstNonEmpty(cfg.OrganizationID.ValueString(), os.Getenv("ANTHROPIC_ORGANIZATION_ID"))
	svcID := firstNonEmpty(cfg.ServiceAccountID.ValueString(), os.Getenv("ANTHROPIC_SERVICE_ACCOUNT_ID"))

	wifCfg, wifErr := auth.NewWIFConfig(ruleID, orgID, svcID)

	data := &providerData{
		adminKey: adminKey,
		wif:      wifCfg,
		wifErr:   wifErr,
	}
	resp.DataSourceData = data
	resp.ResourceData = data

	if wifCfg != nil {
		tflog.Info(ctx, "provider configured with WIF", map[string]any{
			"federation_rule_id":  wifCfg.FederationRuleID,
			"service_account_id": wifCfg.ServiceAccountID,
		})
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (p *anthropicProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *anthropicProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource,
		NewWIFAgentResource,
		NewWIFEnvironmentResource,
		NewWIFVaultResource,
		NewWIFVaultCredentialResource,
		NewMemoryStoreResource,
	}
}
