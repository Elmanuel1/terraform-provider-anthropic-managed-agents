package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WIFVaultResource struct {
	data *providerData
}

type WIFVaultModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Metadata    types.Map    `tfsdk:"metadata"`
	ForceDelete types.Bool   `tfsdk:"force_delete"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func (m *WIFVaultModel) fill(v client.VaultResponse) {
	m.Id = types.StringValue(v.ID)
	m.DisplayName = types.StringValue(v.DisplayName)
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
	m.ArchivedAt = nullableString(v.ArchivedAt)
	m.Metadata = fillMetadata(v.Metadata)
}

func buildVaultBody(data WIFVaultModel) map[string]any {
	body := map[string]any{"display_name": data.DisplayName.ValueString()}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		meta := make(map[string]string)
		data.Metadata.ElementsAs(context.Background(), &meta, false)
		body["metadata"] = meta
	}
	return body
}

func NewWIFVaultResource() resource.Resource {
	return &WIFVaultResource{}
}

var _ resource.Resource = &WIFVaultResource{}
var _ resource.ResourceWithImportState = &WIFVaultResource{}

func (r *WIFVaultResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (r *WIFVaultResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic vault for storing credentials.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workspace_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "ID of the workspace this vault belongs to.",
			},
			"display_name": schema.StringAttribute{
				Required: true,
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Arbitrary string key-value pairs attached to the vault.",
			},
			"force_delete": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, permanently deletes the vault on destroy. When false (default), archives it instead.",
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

func (r *WIFVaultResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WIFVaultResource) requireWIF(diags interface{ AddError(string, string) }) bool {
	if r.data == nil || r.data.wif == nil {
		if r.data != nil && r.data.wifErr != nil {
			diags.AddError("Invalid WIF configuration", r.data.wifErr.Error())
		} else {
			diags.AddError("Missing WIF configuration",
				"Set federation_rule_id, organization_id, service_account_id in the provider block (or via ANTHROPIC_FEDERATION_RULE_ID, ANTHROPIC_ORGANIZATION_ID, ANTHROPIC_SERVICE_ACCOUNT_ID) and ensure TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC is injected.")
		}
		return false
	}
	return true
}

func (r *WIFVaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WIFVaultModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewVaultClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	v, err := c.Create(ctx, buildVaultBody(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create vault: %s", err))
		return
	}
	data.fill(*v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFVaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WIFVaultModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewVaultClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	v, err := c.Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault: %s", err))
		return
	}
	if v == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fill(*v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFVaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WIFVaultModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewVaultClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	v, err := c.Update(ctx, data.Id.ValueString(), buildVaultBody(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update vault: %s", err))
		return
	}
	data.fill(*v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFVaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WIFVaultModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewVaultClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	if data.ForceDelete.ValueBool() {
		if err := c.Delete(ctx, data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete vault: %s", err))
		}
	} else {
		if err := c.Archive(ctx, data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive vault: %s", err))
		}
	}
}

func (r *WIFVaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/vault_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
