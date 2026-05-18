package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VaultCredentialDataSource struct {
	data *providerData
}

type VaultCredentialDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	VaultId     types.String `tfsdk:"vault_id"`
	DisplayName types.String `tfsdk:"display_name"`
	// Non-secret auth fields only — write-only secrets are never returned by the API
	AuthType              types.String `tfsdk:"auth_type"`
	MCPServerURL          types.String `tfsdk:"mcp_server_url"`
	ExpiresAt             types.String `tfsdk:"expires_at"`
	TokenEndpoint         types.String `tfsdk:"token_endpoint"`
	ClientID              types.String `tfsdk:"client_id"`
	TokenEndpointAuthType types.String `tfsdk:"token_endpoint_auth_type"`
	Scope                 types.String `tfsdk:"scope"`
	Resource              types.String `tfsdk:"resource"`
	Metadata              types.Map    `tfsdk:"metadata"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
	ArchivedAt            types.String `tfsdk:"archived_at"`
}

func (m *VaultCredentialDataSourceModel) fill(r client.VaultCredentialResponse) {
	m.Id = types.StringValue(r.ID)
	m.VaultId = types.StringValue(r.VaultID)
	m.DisplayName = nullableString(r.DisplayName)
	m.AuthType = types.StringValue(r.Auth.Type)
	if r.Auth.MCPServerURL != nil {
		m.MCPServerURL = types.StringValue(strings.TrimRight(*r.Auth.MCPServerURL, "/"))
	} else {
		m.MCPServerURL = types.StringNull()
	}
	if r.Auth.ExpiresAt != nil {
		m.ExpiresAt = types.StringValue(*r.Auth.ExpiresAt)
	} else {
		m.ExpiresAt = types.StringNull()
	}
	if r.Auth.Refresh != nil {
		m.TokenEndpoint = types.StringValue(r.Auth.Refresh.TokenEndpoint)
		m.ClientID = types.StringValue(r.Auth.Refresh.ClientID)
		if r.Auth.Refresh.TokenEndpointAuth != nil {
			m.TokenEndpointAuthType = types.StringValue(r.Auth.Refresh.TokenEndpointAuth.Type)
		} else {
			m.TokenEndpointAuthType = types.StringNull()
		}
		m.Scope = nullableString(r.Auth.Refresh.Scope)
		m.Resource = nullableString(r.Auth.Refresh.Resource)
	} else {
		m.TokenEndpoint = types.StringNull()
		m.ClientID = types.StringNull()
		m.TokenEndpointAuthType = types.StringNull()
		m.Scope = types.StringNull()
		m.Resource = types.StringNull()
	}
	m.CreatedAt = types.StringValue(r.CreatedAt)
	m.UpdatedAt = types.StringValue(r.UpdatedAt)
	m.ArchivedAt = nullableString(r.ArchivedAt)
	m.Metadata = fillMetadata(r.Metadata)
}

func NewVaultCredentialDataSource() datasource.DataSource {
	return &VaultCredentialDataSource{}
}

var _ datasource.DataSource = &VaultCredentialDataSource{}
var _ datasource.DataSourceWithConfigure = &VaultCredentialDataSource{}

func (d *VaultCredentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault_credential"
}

func (d *VaultCredentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic vault credential by vault ID and credential ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Credential ID.",
			},
			"vault_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the vault this credential belongs to.",
			},
			"workspace_id": schema.StringAttribute{
				Optional:    true,
				Description: "Workspace ID. Required when using WIF authentication.",
			},
			"display_name":            schema.StringAttribute{Computed: true},
			"auth_type":               schema.StringAttribute{Computed: true},
			"mcp_server_url":          schema.StringAttribute{Computed: true},
			"expires_at":              schema.StringAttribute{Computed: true},
			"token_endpoint":          schema.StringAttribute{Computed: true},
			"client_id":               schema.StringAttribute{Computed: true},
			"token_endpoint_auth_type": schema.StringAttribute{Computed: true},
			"scope":                   schema.StringAttribute{Computed: true},
			"resource":                schema.StringAttribute{Computed: true},
			"metadata": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *VaultCredentialDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.data = data
}

func (d *VaultCredentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VaultCredentialDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := resolveWorkspaceCredentials(ctx, d.data, "data.anthropic_vault_credential", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cred, err := client.NewVaultCredentialClient(creds).Read(ctx, data.VaultId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault credential: %s", err))
		return
	}
	if cred == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Vault credential %q not found in vault %q.", data.Id.ValueString(), data.VaultId.ValueString()))
		return
	}
	data.fill(*cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
