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

type MemoryStoreDataSource struct {
	data *providerData
}

type MemoryStoreDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	Type        types.String `tfsdk:"type"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Metadata    types.Map    `tfsdk:"metadata"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func (m *MemoryStoreDataSourceModel) fill(s client.MemoryStoreResponse) {
	m.Id = types.StringValue(s.ID)
	m.Type = types.StringValue(s.Type)
	m.Name = types.StringValue(s.Name)
	m.Description = nullableString(s.Description)
	m.CreatedAt = types.StringValue(s.CreatedAt)
	m.UpdatedAt = types.StringValue(s.UpdatedAt)
	m.ArchivedAt = nullableString(s.ArchivedAt)
	m.Metadata = fillMetadata(s.Metadata)
}

func NewMemoryStoreDataSource() datasource.DataSource {
	return &MemoryStoreDataSource{}
}

var _ datasource.DataSource = &MemoryStoreDataSource{}
var _ datasource.DataSourceWithConfigure = &MemoryStoreDataSource{}

func (d *MemoryStoreDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_memory_store"
}

func (d *MemoryStoreDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Anthropic memory store by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Memory store ID.",
			},
			"type":        schema.StringAttribute{Computed: true, Description: "Memory store type (e.g. knowledge_base)."},
			"name":        schema.StringAttribute{Computed: true},
			"description": schema.StringAttribute{Computed: true},
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

func (d *MemoryStoreDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MemoryStoreDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MemoryStoreDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.data == nil || d.data.adminKey == "" {
		resp.Diagnostics.AddError(
			"Missing admin API key",
			"Set admin_api_key in the provider block. Required for anthropic_memory_store data source.",
		)
		return
	}

	c := client.NewMemoryStoreClient(auth.WithBeta(auth.AdminAPIKey{Key: d.data.adminKey}, auth.AgentsBeta))
	store, err := c.Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read memory store: %s", err))
		return
	}
	if store == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Memory store %q not found.", data.Id.ValueString()))
		return
	}
	data.fill(*store)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
