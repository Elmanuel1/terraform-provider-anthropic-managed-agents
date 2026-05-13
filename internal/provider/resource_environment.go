package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentResource struct {
	data *providerData
}

type EnvironmentModel struct {
	Id                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	NetworkingType       types.String `tfsdk:"networking_type"`
	AllowedHosts         types.List   `tfsdk:"allowed_hosts"`
	AllowMCPServers      types.Bool   `tfsdk:"allow_mcp_servers"`
	AllowPackageManagers types.Bool   `tfsdk:"allow_package_managers"`
	Packages             types.String `tfsdk:"packages"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	ArchivedAt           types.String `tfsdk:"archived_at"`
}

type environmentAPIResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Config *struct {
		Packages   map[string][]string `json:"packages"`
		Networking *struct {
			Type                 string   `json:"type"`
			AllowedHosts         []string `json:"allowed_hosts"`
			AllowMCPServers      *bool    `json:"allow_mcp_servers"`
			AllowPackageManagers *bool    `json:"allow_package_managers"`
		} `json:"networking"`
	} `json:"config"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	ArchivedAt *string `json:"archived_at"`
}

func nullableBool(b *bool) types.Bool {
	if b == nil {
		return types.BoolValue(false)
	}
	return types.BoolValue(*b)
}

func (m *EnvironmentModel) fill(e environmentAPIResponse) {
	m.Id = types.StringValue(e.ID)
	m.Name = types.StringValue(e.Name)
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

	if e.Config != nil && len(e.Config.Packages) > 0 {
		b, err := json.Marshal(e.Config.Packages)
		if err == nil {
			m.Packages = types.StringValue(string(b))
		}
	} else {
		m.Packages = types.StringNull()
	}
}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}

func (r *EnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic cloud environment for agent sessions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"networking_type": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("unrestricted"),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "unrestricted (default) or limited.",
			},
			"allowed_hosts": schema.ListAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				Default:       listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
				Description:   "Allowed outbound hosts when networking_type is limited.",
			},
			"allow_mcp_servers": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
				Description:   "Allow MCP server network access. Only applies when networking_type is limited.",
			},
			"allow_package_managers": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
				Description:   "Allow package manager network access (PyPI, npm, etc). Only applies when networking_type is limited.",
			},
			"packages": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   `JSON-encoded packages to pre-install. Example: {"pip":["pandas","numpy"],"npm":["express"]}. Supported managers: apt, cargo, gem, go, npm, pip.`,
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at":  schema.StringAttribute{Computed: true},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *EnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EnvironmentResource) buildConfig(ctx context.Context, data EnvironmentModel) map[string]any {
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

	return config
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":   data.Name.ValueString(),
		"config": r.buildConfig(ctx, data),
	}

	raw, status, err := doRequest(ctx, r.data, http.MethodPost, "/v1/environments", body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment: %s", err))
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create environment, status %d: %s", status, raw))
		return
	}

	var e environmentAPIResponse
	if err := json.Unmarshal(raw, &e); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse environment response: %s", err))
		return
	}
	data.fill(e)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, status, err := doRequest(ctx, r.data, http.MethodGet, "/v1/environments/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment: %s", err))
		return
	}
	if status == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, status %d: %s", status, raw))
		return
	}

	var e environmentAPIResponse
	if err := json.Unmarshal(raw, &e); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse environment response: %s", err))
		return
	}
	data.fill(e)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes carry RequiresReplace; Update is never called.
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, status, err := doRequest(ctx, r.data, http.MethodDelete, "/v1/environments/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment: %s", err))
		return
	}
	if status != http.StatusOK && status != http.StatusNoContent && status != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete environment, status %d", status))
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
