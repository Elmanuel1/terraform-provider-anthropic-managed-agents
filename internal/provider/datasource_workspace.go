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

type WorkspaceDataSource struct {
	data *providerData
}

type WorkspaceDataModel struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CreatedAt  types.String `tfsdk:"created_at"`
	ArchivedAt types.String `tfsdk:"archived_at"`
}

func NewWorkspaceDataSource() datasource.DataSource {
	return &WorkspaceDataSource{}
}

var _ datasource.DataSource = &WorkspaceDataSource{}
var _ datasource.DataSourceWithConfigure = &WorkspaceDataSource{}

func (d *WorkspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (d *WorkspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic workspace by ID or name. Requires admin_api_key in the provider block.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Workspace ID (wrks_...). One of id or name must be set.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Workspace name. One of id or name must be set.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 creation timestamp.",
			},
			"archived_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 archival timestamp, or null if active.",
			},
		},
	}
}

func (d *WorkspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WorkspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkspaceDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError("Missing attribute", "One of id or name must be set.")
		return
	}
	if !data.Id.IsNull() && !data.Name.IsNull() {
		resp.Diagnostics.AddError("Conflicting attributes", "Only one of id or name may be set.")
		return
	}

	if d.data == nil || d.data.adminKey == "" {
		resp.Diagnostics.AddError("Missing admin_api_key", "admin_api_key must be set in the provider block to use data.anthropic_workspace.")
		return
	}

	c := client.NewWorkspaceClient(auth.AdminAPIKey{Key: d.data.adminKey})

	id := data.Id.ValueString()
	if id == "" {
		var err error
		id, err = c.ResolveByName(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to resolve workspace by name: %s", err))
			return
		}
	}

	w, err := c.Read(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workspace: %s", err))
		return
	}
	if w == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Workspace %q not found.", id))
		return
	}

	data.Id = types.StringValue(w.ID)
	data.Name = types.StringValue(w.Name)
	data.CreatedAt = types.StringValue(w.CreatedAt)
	data.ArchivedAt = nullableString(w.ArchivedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
