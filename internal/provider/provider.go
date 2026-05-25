package provider

import (
	"context"
	"os"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
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
	WorkspaceAPIKey  types.String `tfsdk:"workspace_api_key"`
	FederationRuleID types.String `tfsdk:"federation_rule_id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
}

type providerData struct {
	adminKey        string
	workspaceAPIKey string
	wif             *auth.WIFConfig
	wifErr          error
}

func (p *anthropicProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anthropic"
}

func (p *anthropicProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Anthropic platform resources. " +
			"All attributes are optional in the provider block. " +
			"Each resource validates only the credentials it needs at operation time.",
		Attributes: map[string]schema.Attribute{
			"admin_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Anthropic Admin API key (sk-ant-admin-...). Required for anthropic_workspace and anthropic_memory_store.",
			},
			"workspace_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Anthropic workspace API key (sk-ant-api03-...). Used for anthropic_agent authentication. When both workspace_api_key and WIF are configured, WIF takes precedence.",
			},
			"federation_rule_id": schema.StringAttribute{
				Optional:    true,
				Description: "Federation rule ID (fdrl_...). Required for anthropic_agent, anthropic_environment, anthropic_vault, anthropic_vault_credential when using WIF.",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Description: "Anthropic organization UUID. Required for anthropic_agent, anthropic_environment, anthropic_vault, anthropic_vault_credential when using WIF.",
			},
			"service_account_id": schema.StringAttribute{
				Optional:    true,
				Description: "Service account ID (svac_...). Required for anthropic_agent, anthropic_environment, anthropic_vault, anthropic_vault_credential when using WIF.",
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

	adminKey := cfg.AdminAPIKey.ValueString()
	if adminKey == "" {
		adminKey = os.Getenv("ANTHROPIC_ADMIN_API_KEY")
	}
	workspaceAPIKey := cfg.WorkspaceAPIKey.ValueString()
	if workspaceAPIKey == "" {
		workspaceAPIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	ruleID := cfg.FederationRuleID.ValueString()
	orgID := cfg.OrganizationID.ValueString()
	svcID := cfg.ServiceAccountID.ValueString()

	wifCfg, wifErr := auth.NewWIFConfig(ruleID, orgID, svcID)

	data := &providerData{
		adminKey:        adminKey,
		workspaceAPIKey: workspaceAPIKey,
		wif:             wifCfg,
		wifErr:          wifErr,
	}
	resp.DataSourceData = data
	resp.ResourceData = data

	if wifCfg != nil {
		tflog.Info(ctx, "provider configured with WIF", map[string]any{
			"federation_rule_id":  wifCfg.FederationRuleID,
			"service_account_id": wifCfg.ServiceAccountID,
		})
	} else if wifErr != nil {
		tflog.Warn(ctx, "partial WIF configuration provided but incomplete — WIF resources will fail at apply time",
			map[string]any{"error": wifErr.Error()})
	}
}


func (p *anthropicProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewWorkspaceDataSource,
		NewAgentDataSource,
		NewEnvironmentDataSource,
		NewVaultDataSource,
		NewVaultCredentialDataSource,
		NewMemoryStoreDataSource,
		NewSkillDataSource,
	}
}

func (p *anthropicProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource,
		NewWIFAgentResource,
		NewWIFEnvironmentResource,
		NewWIFVaultResource,
		NewWIFVaultCredentialResource,
		NewMemoryStoreResource,
		NewSkillResource,
	}
}
