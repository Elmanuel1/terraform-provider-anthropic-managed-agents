package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AgentResource struct {
	data *providerData
}

type AgentModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	Name        types.String `tfsdk:"name"`
	Model       types.String `tfsdk:"model"`
	ModelSpeed  types.String `tfsdk:"model_speed"`
	System      types.String `tfsdk:"system"`
	Description types.String `tfsdk:"description"`
	Tools       types.String `tfsdk:"tools"`
	MCPServers  types.String `tfsdk:"mcp_servers"`
	Skills      types.String `tfsdk:"skills"`
	Multiagent  types.String `tfsdk:"multiagent"`
	Metadata    types.Map    `tfsdk:"metadata"`
	Version     types.Int64  `tfsdk:"version"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

func (m *AgentModel) fill(a client.AgentResponse) {
	m.Id = types.StringValue(a.ID)
	m.Name = types.StringValue(a.Name)
	m.Model = types.StringValue(a.Model.ID)
	m.ModelSpeed = types.StringValue(a.Model.Speed)
	m.Version = types.Int64Value(int64(a.Version))
	m.CreatedAt = types.StringValue(a.CreatedAt)
	m.UpdatedAt = types.StringValue(a.UpdatedAt)
	m.System = nullableString(a.System)
	m.Description = nullableString(a.Description)
	m.ArchivedAt = nullableString(a.ArchivedAt)
	m.Tools = marshalJSONList(a.Tools)
	m.MCPServers = marshalJSONList(a.MCPServers)
	m.Skills = marshalJSONList(a.Skills)
	if a.Multiagent != nil && string(*a.Multiagent) != "null" {
		m.Multiagent = types.StringValue(string(*a.Multiagent))
	} else {
		m.Multiagent = types.StringNull()
	}
	m.Metadata = fillMetadata(a.Metadata)
}

func buildAgentBody(data AgentModel) (map[string]any, error) {
	body := map[string]any{
		"name": data.Name.ValueString(),
		"model": map[string]string{
			"id":    data.Model.ValueString(),
			"speed": data.ModelSpeed.ValueString(),
		},
	}
	if !data.System.IsNull() && !data.System.IsUnknown() {
		body["system"] = data.System.ValueString()
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}
	if !data.Tools.IsNull() && !data.Tools.IsUnknown() {
		var tools []interface{}
		if err := json.Unmarshal([]byte(data.Tools.ValueString()), &tools); err != nil {
			return nil, fmt.Errorf("invalid tools JSON: %w", err)
		}
		body["tools"] = tools
	}
	if !data.MCPServers.IsNull() && !data.MCPServers.IsUnknown() {
		var mcpServers []interface{}
		if err := json.Unmarshal([]byte(data.MCPServers.ValueString()), &mcpServers); err != nil {
			return nil, fmt.Errorf("invalid mcp_servers JSON: %w", err)
		}
		body["mcp_servers"] = mcpServers
	}
	if !data.Skills.IsNull() && !data.Skills.IsUnknown() {
		var skills []interface{}
		if err := json.Unmarshal([]byte(data.Skills.ValueString()), &skills); err != nil {
			return nil, fmt.Errorf("invalid skills JSON: %w", err)
		}
		body["skills"] = skills
	}
	if !data.Multiagent.IsNull() && !data.Multiagent.IsUnknown() {
		var multiagent interface{}
		if err := json.Unmarshal([]byte(data.Multiagent.ValueString()), &multiagent); err != nil {
			return nil, fmt.Errorf("invalid multiagent JSON: %w", err)
		}
		body["multiagent"] = multiagent
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() && len(data.Metadata.Elements()) > 0 {
		meta := make(map[string]string, len(data.Metadata.Elements()))
		data.Metadata.ElementsAs(context.Background(), &meta, false)
		body["metadata"] = meta
	}
	return body, nil
}

func NewAgentResource() resource.Resource {
	return &AgentResource{}
}

var _ resource.Resource = &AgentResource{}
var _ resource.ResourceWithImportState = &AgentResource{}
var _ resource.ResourceWithModifyPlan = &AgentResource{}

func (r *AgentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wif_agent"
}

func (r *AgentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic agent.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workspace_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "ID of the workspace this agent belongs to.",
			},
			"name": schema.StringAttribute{Required: true},
			"model": schema.StringAttribute{
				Required:    true,
				Description: "Model ID, e.g. claude-opus-4-7 or claude-sonnet-4-6.",
			},
			"model_speed": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("standard"),
				Description: "Inference speed: standard (default) or fast.",
			},
			"system":      schema.StringAttribute{Optional: true, Computed: true},
			"description": schema.StringAttribute{Optional: true, Computed: true},
			"tools": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   `JSON-encoded tools array. Example: [{"type":"agent_toolset_20260401"}]`,
			},
			"mcp_servers": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   `JSON-encoded MCP servers array. Example: [{"name":"my-server","type":"url","url":"https://..."}]. Maximum 20, names must be unique.`,
			},
			"skills": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   `JSON-encoded skills array. Example: [{"type":"anthropic","skill_id":"xlsx"}]. Maximum 20.`,
			},
			"multiagent": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   `JSON-encoded multiagent coordinator config. Example: {"type":"coordinator","agents":["agent_id_1","agent_id_2"]}.`,
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Arbitrary string key-value pairs attached to the agent.",
			},
			"version": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"archived_at": schema.StringAttribute{Computed: true},
		},
	}
}

// ModifyPlan marks server-managed fields (version, updated_at) as Unknown only
// when an update is actually happening, preventing spurious no-op update plans.
func (r *AgentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy or create — nothing to do.
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var plan, state AgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !agentUserFieldsChanged(plan, state) {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("version"), types.Int64Unknown())...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("updated_at"), types.StringUnknown())...)
}

// agentUserFieldsChanged reports whether any user-controlled field differs between plan and state.
func agentUserFieldsChanged(plan, state AgentModel) bool {
	return !plan.Name.Equal(state.Name) ||
		!plan.System.Equal(state.System) ||
		!plan.Description.Equal(state.Description) ||
		!plan.Model.Equal(state.Model) ||
		!plan.ModelSpeed.Equal(state.ModelSpeed) ||
		!plan.Tools.Equal(state.Tools) ||
		!plan.MCPServers.Equal(state.MCPServers) ||
		!plan.Skills.Equal(state.Skills) ||
		!plan.Multiagent.Equal(state.Multiagent) ||
		!plan.Metadata.Equal(state.Metadata)
}

func (r *AgentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AgentResource) requireWIF(diags interface{ AddError(string, string) }) bool {
	if r.data == nil || r.data.wif == nil {
		if r.data != nil && r.data.wifErr != nil {
			diags.AddError("Invalid WIF configuration", r.data.wifErr.Error())
		} else {
			diags.AddError("Missing WIF configuration",
				"ANTHROPIC_FEDERATION_RULE_ID, ANTHROPIC_ORGANIZATION_ID, ANTHROPIC_SERVICE_ACCOUNT_ID, and one of TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC or TFC_WORKLOAD_IDENTITY_TOKEN are required for agent resources.")
		}
		return false
	}
	return true
}

func (r *AgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	body, err := buildAgentBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent configuration", err.Error())
		return
	}
	c := client.NewAgentClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	agent, err := c.Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create agent: %s", err))
		return
	}
	data.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewAgentClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	agent, err := c.Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}
	if agent == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Read version from prior state — the plan marks version Unknown so the API's
	// incremented value is accepted, but we still need the current version number
	// for the optimistic-lock field in the update request body.
	var state AgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	body, err := buildAgentBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent configuration", err.Error())
		return
	}
	body["version"] = state.Version.ValueInt64()

	c := client.NewAgentClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	agent, err := c.Update(ctx, data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update agent: %s", err))
		return
	}
	data.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewAgentClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	if err := c.Delete(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive agent: %s", err))
	}
}

func (r *AgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/agent_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
