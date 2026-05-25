package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VaultDataSource struct {
	data *providerData
}

type VaultDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Metadata    types.Map    `tfsdk:"metadata"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func (m *VaultDataSourceModel) fill(v client.VaultResponse) {
	m.Id = types.StringValue(v.ID)
	m.DisplayName = types.StringValue(v.DisplayName)
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
	m.ArchivedAt = nullableString(v.ArchivedAt)
	m.Metadata = fillMetadata(v.Metadata)
}

func NewVaultDataSource() datasource.DataSource {
	return &VaultDataSource{}
}

var _ datasource.DataSource = &VaultDataSource{}
var _ datasource.DataSourceWithConfigure = &VaultDataSource{}

func (d *VaultDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (d *VaultDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic vault by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Vault ID.",
			},
			"workspace_id": schema.StringAttribute{
				Optional:    true,
				Description: "Workspace ID. Required when using WIF authentication.",
			},
			"display_name": schema.StringAttribute{Computed: true},
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

func (d *VaultDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VaultDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VaultDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := resolveWorkspaceCredentials(ctx, d.data, "data.anthropic_vault", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	vault, err := client.NewVaultClient(auth.WithBeta(creds, auth.AgentsBeta)).Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault: %s", err))
		return
	}
	if vault == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Vault %q not found.", data.Id.ValueString()))
		return
	}
	data.fill(*vault)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
