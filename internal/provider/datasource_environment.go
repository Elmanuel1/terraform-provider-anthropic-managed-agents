package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentDataSource struct {
	data *providerData
}

type EnvironmentDataModel struct {
	Id                   types.String `tfsdk:"id"`
	WorkspaceId          types.String `tfsdk:"workspace_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	NetworkingType       types.String `tfsdk:"networking_type"`
	AllowedHosts         types.List   `tfsdk:"allowed_hosts"`
	AllowMCPServers      types.Bool   `tfsdk:"allow_mcp_servers"`
	AllowPackageManagers types.Bool   `tfsdk:"allow_package_managers"`
	Packages             jsontypes.Normalized `tfsdk:"packages"`
	Metadata             types.Map    `tfsdk:"metadata"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	ArchivedAt           types.String `tfsdk:"archived_at"`
}

func (m *EnvironmentDataModel) fill(e client.EnvironmentResponse) error {
	m.Id = types.StringValue(e.ID)
	m.Name = types.StringValue(e.Name)
	m.Description = nullableString(e.Description)
	m.CreatedAt = types.StringValue(e.CreatedAt)
	m.UpdatedAt = types.StringValue(e.UpdatedAt)
	m.ArchivedAt = nullableString(e.ArchivedAt)
	m.Metadata = fillMetadata(e.Metadata)

	emptyHosts := types.ListValueMust(types.StringType, []attr.Value{})
	if e.Config == nil || e.Config.Networking == nil {
		m.NetworkingType = types.StringValue("unrestricted")
		m.AllowedHosts = emptyHosts
		m.AllowMCPServers = types.BoolValue(false)
		m.AllowPackageManagers = types.BoolValue(false)
	} else {
		n := e.Config.Networking
		m.NetworkingType = types.StringValue(n.Type)
		if n.Type == "limited" {
			m.AllowMCPServers = nullableBool(n.AllowMCPServers)
			m.AllowPackageManagers = nullableBool(n.AllowPackageManagers)
			if len(n.AllowedHosts) > 0 {
				vals := make([]attr.Value, len(n.AllowedHosts))
				for i, h := range n.AllowedHosts {
					vals[i] = types.StringValue(h)
				}
				hosts, _ := types.ListValue(types.StringType, vals)
				m.AllowedHosts = hosts
			} else {
				m.AllowedHosts = emptyHosts
			}
		} else {
			m.AllowedHosts = emptyHosts
			m.AllowMCPServers = types.BoolValue(false)
			m.AllowPackageManagers = types.BoolValue(false)
		}
	}
	if e.Config != nil {
		pkgs, err := normalizePackages(e.Config.Packages)
		if err != nil {
			return fmt.Errorf("marshaling packages: %w", err)
		}
		m.Packages = pkgs
	} else {
		m.Packages = jsontypes.NewNormalizedNull()
	}
	return nil
}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

var _ datasource.DataSource = &EnvironmentDataSource{}
var _ datasource.DataSourceWithConfigure = &EnvironmentDataSource{}

func (d *EnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic environment by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Required: true, Description: "Environment ID."},
			"workspace_id":           schema.StringAttribute{Optional: true, Description: "Workspace ID. Required when using WIF authentication."},
			"name":                   schema.StringAttribute{Computed: true},
			"description":            schema.StringAttribute{Computed: true},
			"networking_type":        schema.StringAttribute{Computed: true},
			"allowed_hosts":          schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"allow_mcp_servers":      schema.BoolAttribute{Computed: true},
			"allow_package_managers": schema.BoolAttribute{Computed: true},
			"packages":               schema.StringAttribute{Computed: true, CustomType: jsontypes.NormalizedType{}, Description: "JSON-encoded packages map."},
			"metadata":               schema.MapAttribute{Computed: true, ElementType: types.StringType},
			"created_at":             schema.StringAttribute{Computed: true},
			"updated_at":             schema.StringAttribute{Computed: true},
			"archived_at":            schema.StringAttribute{Computed: true},
		},
	}
}

func (d *EnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.data = pd
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	creds := resolveWorkspaceCredentials(ctx, d.data, "data.anthropic_environment", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	e, err := client.NewEnvironmentClient(creds).Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment: %s", err))
		return
	}
	if e == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Environment %q not found.", data.Id.ValueString()))
		return
	}
	if err := data.fill(*e); err != nil {
		resp.Diagnostics.AddError("Internal Error", fmt.Sprintf("marshaling environment response: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
