package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WorkspaceResource struct {
	data *providerData
}

type WorkspaceModel struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	CreatedAt  types.String `tfsdk:"created_at"`
	ArchivedAt types.String `tfsdk:"archived_at"`
}

func (m *WorkspaceModel) fill(w client.WorkspaceResponse) {
	m.Id = types.StringValue(w.ID)
	m.Name = types.StringValue(w.Name)
	m.CreatedAt = types.StringValue(w.CreatedAt)
	m.ArchivedAt = nullableString(w.ArchivedAt)
}

func NewWorkspaceResource() resource.Resource {
	return &WorkspaceResource{}
}

var _ resource.Resource = &WorkspaceResource{}
var _ resource.ResourceWithImportState = &WorkspaceResource{}

func (r *WorkspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (r *WorkspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Workspace name as it appears in the Anthropic Console.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *WorkspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkspaceResource) requireAdminKey(diags interface{ AddError(string, string) }) bool {
	if r.data == nil || r.data.adminKey == "" {
		diags.AddError("Missing admin API key",
			"Set admin_api_key in the provider block or ANTHROPIC_ADMIN_API_KEY environment variable. Required for anthropic_workspace.")
		return false
	}
	return true
}

func (r *WorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var data WorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.NewWorkspaceClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AdminBeta))
	w, err := c.Create(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workspace: %s", err))
		return
	}
	data.fill(*w)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var data WorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.NewWorkspaceClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AdminBeta))
	w, err := c.Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workspace: %s", err))
		return
	}
	if w == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fill(*w)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var data WorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.NewWorkspaceClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AdminBeta))
	w, err := c.Update(ctx, data.Id.ValueString(), map[string]any{"name": data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update workspace: %s", err))
		return
	}
	data.fill(*w)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var data WorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.NewWorkspaceClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AdminBeta))
	if err := c.Archive(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive workspace: %s", err))
	}
}

func (r *WorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	c := client.NewWorkspaceClient(auth.WithBeta(auth.AdminAPIKey{Key: r.data.adminKey}, auth.AdminBeta))
	id, err := c.ResolveByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to resolve workspace %q: %s", req.ID, err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
