package provider

import (
	"context"
	"fmt"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type MemoryStoreResource struct {
	data *providerData
}

type MemoryStoreModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Metadata    types.Map    `tfsdk:"metadata"`
	ForceDelete types.Bool   `tfsdk:"force_delete"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func (m *MemoryStoreModel) fill(s client.MemoryStoreResponse) {
	m.Id = types.StringValue(s.ID)
	m.Name = types.StringValue(s.Name)
	m.Description = nullableString(s.Description)
	m.CreatedAt = types.StringValue(s.CreatedAt)
	m.UpdatedAt = types.StringValue(s.UpdatedAt)
	m.ArchivedAt = nullableString(s.ArchivedAt)
	m.Metadata = fillMetadata(s.Metadata)
}

func buildMemoryStoreBody(ctx context.Context, data MemoryStoreModel, diags *diag.Diagnostics) map[string]any {
	body := map[string]any{"name": data.Name.ValueString()}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		meta := make(map[string]string)
		diags.Append(data.Metadata.ElementsAs(ctx, &meta, false)...)
		body["metadata"] = meta
	}
	return body
}

func NewMemoryStoreResource() resource.Resource {
	return &MemoryStoreResource{}
}

var _ resource.Resource = &MemoryStoreResource{}
var _ resource.ResourceWithImportState = &MemoryStoreResource{}

func (r *MemoryStoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_memory_store"
}

func (r *MemoryStoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic memory store.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Arbitrary string key-value pairs attached to the memory store.",
			},
			"force_delete": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, permanently deletes the memory store on destroy. When false (default), archives it instead.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *MemoryStoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.data = data
}

func (r *MemoryStoreResource) requireAdminKey(diags interface{ AddError(string, string) }) bool {
	if r.data == nil || r.data.adminKey == "" {
		diags.AddError("Missing admin API key",
			"Set admin_api_key in the provider block or ANTHROPIC_ADMIN_API_KEY environment variable. Required for anthropic_memory_store.")
		return false
	}
	return true
}

func (r *MemoryStoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MemoryStoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}

	c := client.NewMemoryStoreClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AgentsBeta))
	s, err := c.Create(ctx, buildMemoryStoreBody(ctx, data, &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create memory store: %s", err))
		return
	}
	data.fill(*s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemoryStoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MemoryStoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}

	c := client.NewMemoryStoreClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AgentsBeta))
	s, err := c.Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read memory store: %s", err))
		return
	}
	if s == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fill(*s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemoryStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MemoryStoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}

	c := client.NewMemoryStoreClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AgentsBeta))
	s, err := c.Update(ctx, data.Id.ValueString(), buildMemoryStoreBody(ctx, data, &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update memory store: %s", err))
		return
	}
	data.fill(*s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemoryStoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MemoryStoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}

	c := client.NewMemoryStoreClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AgentsBeta))
	if data.ForceDelete.ValueBool() {
		if err := c.Delete(ctx, data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete memory store: %s", err))
		}
	} else {
		if err := c.Archive(ctx, data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive memory store: %s", err))
		}
	}
}

func (r *MemoryStoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
