package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WIFEnvironmentResource struct {
	data *providerData
}

type WIFEnvironmentModel struct {
	Id                   types.String `tfsdk:"id"`
	WorkspaceId          types.String `tfsdk:"workspace_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	NetworkingType       types.String `tfsdk:"networking_type"`
	AllowedHosts         types.List   `tfsdk:"allowed_hosts"`
	AllowMCPServers      types.Bool   `tfsdk:"allow_mcp_servers"`
	AllowPackageManagers types.Bool   `tfsdk:"allow_package_managers"`
	Packages             types.String `tfsdk:"packages"`
	Metadata             types.Map    `tfsdk:"metadata"`
	ForceDelete          types.Bool   `tfsdk:"force_delete"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	ArchivedAt           types.String `tfsdk:"archived_at"`
}

func nullableBool(b *bool) types.Bool {
	if b == nil {
		return types.BoolValue(false)
	}
	return types.BoolValue(*b)
}

func (m *WIFEnvironmentModel) fill(e client.EnvironmentResponse) {
	m.Id = types.StringValue(e.ID)
	m.Name = types.StringValue(e.Name)
	m.Description = nullableString(e.Description)
	m.CreatedAt = types.StringValue(e.CreatedAt)
	m.UpdatedAt = types.StringValue(e.UpdatedAt)
	m.ArchivedAt = nullableString(e.ArchivedAt)

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
			if len(n.AllowedHosts) == 0 {
				m.AllowedHosts = emptyHosts
			} else {
				vals := make([]attr.Value, len(n.AllowedHosts))
				for i, h := range n.AllowedHosts {
					vals[i] = types.StringValue(h)
				}
				hosts, _ := types.ListValue(types.StringType, vals)
				m.AllowedHosts = hosts
			}
		} else {
			m.AllowedHosts = emptyHosts
			m.AllowMCPServers = types.BoolValue(false)
			m.AllowPackageManagers = types.BoolValue(false)
		}
	}

	if e.Config != nil {
		m.Packages = normalizePackages(e.Config.Packages)
	} else {
		m.Packages = types.StringNull()
	}

	m.Metadata = fillMetadata(e.Metadata)
}

func NewWIFEnvironmentResource() resource.Resource {
	return &WIFEnvironmentResource{}
}

var _ resource.Resource = &WIFEnvironmentResource{}
var _ resource.ResourceWithImportState = &WIFEnvironmentResource{}

func (r *WIFEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *WIFEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic cloud environment for agent sessions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workspace_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "ID of the workspace this environment belongs to.",
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"networking_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("unrestricted"),
				Description: "unrestricted (default) or limited.",
			},
			"allowed_hosts": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "Allowed outbound hosts when networking_type is limited.",
			},
			"allow_mcp_servers": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allow MCP server network access. Only applies when networking_type is limited.",
			},
			"allow_package_managers": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allow package manager network access (PyPI, npm, etc). Only applies when networking_type is limited.",
			},
			"packages": schema.StringAttribute{
				Optional:    true,
				Description: `JSON-encoded packages to pre-install. Example: {"pip":["pandas","numpy"],"npm":["express"]}. Supported managers: apt, cargo, gem, go, npm, pip.`,
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Arbitrary string key-value pairs attached to the environment.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"force_delete": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, permanently deletes the environment on destroy. When false (default), archives it instead.",
			},
			"updated_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *WIFEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WIFEnvironmentResource) buildBody(ctx context.Context, data WIFEnvironmentModel) map[string]any {
	networking := map[string]any{"type": data.NetworkingType.ValueString()}
	if data.NetworkingType.ValueString() == "limited" {
		var hosts []string
		data.AllowedHosts.ElementsAs(ctx, &hosts, false)
		if len(hosts) > 0 {
			networking["allowed_hosts"] = hosts
		}
		networking["allow_mcp_servers"] = data.AllowMCPServers.ValueBool()
		networking["allow_package_managers"] = data.AllowPackageManagers.ValueBool()
	}

	config := map[string]any{"type": "cloud", "networking": networking}
	if !data.Packages.IsNull() && !data.Packages.IsUnknown() && data.Packages.ValueString() != "" {
		var pkgs map[string]interface{}
		if err := json.Unmarshal([]byte(data.Packages.ValueString()), &pkgs); err == nil && len(pkgs) > 0 {
			config["packages"] = pkgs
		}
	}

	body := map[string]any{
		"name":   data.Name.ValueString(),
		"config": config,
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() && len(data.Metadata.Elements()) > 0 {
		meta := make(map[string]string, len(data.Metadata.Elements()))
		data.Metadata.ElementsAs(ctx, &meta, false)
		body["metadata"] = meta
	}
	return body
}

func (r *WIFEnvironmentResource) creds(workspaceID string) auth.WIFBearer {
	return auth.WIFBearer{Config: r.data.wif, WorkspaceID: workspaceID}
}

func (r *WIFEnvironmentResource) requireWIF(diags interface{ AddError(string, string) }) bool {
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

func (r *WIFEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WIFEnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewEnvironmentClient(r.creds(data.WorkspaceId.ValueString()))
	env, err := c.Create(ctx, r.buildBody(ctx, data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment: %s", err))
		return
	}
	data.fill(*env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WIFEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewEnvironmentClient(r.creds(data.WorkspaceId.ValueString()))
	env, err := c.Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment: %s", err))
		return
	}
	if env == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fill(*env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WIFEnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewEnvironmentClient(r.creds(data.WorkspaceId.ValueString()))
	env, err := c.Update(ctx, data.Id.ValueString(), r.buildBody(ctx, data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update environment: %s", err))
		return
	}
	data.fill(*env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WIFEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewEnvironmentClient(r.creds(data.WorkspaceId.ValueString()))
	if data.ForceDelete.ValueBool() {
		if err := c.Delete(ctx, data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment: %s", err))
		}
	} else {
		if err := c.Archive(ctx, data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive environment: %s", err))
		}
	}
}

func (r *WIFEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/environment_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
