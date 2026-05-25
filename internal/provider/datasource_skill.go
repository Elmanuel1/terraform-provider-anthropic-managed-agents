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

type SkillDataSource struct {
	data *providerData
}

type SkillDataModel struct {
	ID           types.String `tfsdk:"id"`
	WorkspaceId  types.String `tfsdk:"workspace_id"`
	DisplayTitle types.String `tfsdk:"display_title"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (m *SkillDataModel) fill(s client.SkillResponse) {
	m.ID = types.StringValue(s.ID)
	m.DisplayTitle = nullableString(s.DisplayTitle)
	m.CreatedAt = types.StringValue(s.CreatedAt)
	m.UpdatedAt = types.StringValue(s.UpdatedAt)
}

func NewSkillDataSource() datasource.DataSource {
	return &SkillDataSource{}
}

var _ datasource.DataSource = &SkillDataSource{}
var _ datasource.DataSourceWithConfigure = &SkillDataSource{}

func (d *SkillDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

func (d *SkillDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic skill by ID. Supports WIF (workspace_id required) and workspace API key authentication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Skill ID.",
			},
			"workspace_id": schema.StringAttribute{
				Optional:    true,
				Description: "Workspace ID for WIF authentication. When set, WIF is used. When omitted, workspace_api_key is used.",
			},
			"display_title": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable title for the skill.",
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *SkillDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SkillDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SkillDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := resolveWorkspaceCredentials(ctx, d.data, "data.anthropic_skill", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if creds == nil {
		return
	}
	c := client.NewSkillClient(auth.WithBeta(creds, auth.SkillsBeta))
	s, err := c.Read(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read skill: %s", err))
		return
	}
	if s == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Skill %q not found.", data.ID.ValueString()))
		return
	}

	data.fill(*s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
