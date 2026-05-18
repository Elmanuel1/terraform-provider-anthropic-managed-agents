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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type WIFAgentModel struct {
	WorkspaceId types.String `tfsdk:"workspace_id"`
	AgentCoreModel
}

type WIFAgentResource struct {
	data *providerData
}

func NewWIFAgentResource() resource.Resource {
	return &WIFAgentResource{}
}

var _ resource.Resource = &WIFAgentResource{}
var _ resource.ResourceWithImportState = &WIFAgentResource{}
var _ resource.ResourceWithModifyPlan = &WIFAgentResource{}

func (r *WIFAgentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (r *WIFAgentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := agentCoreSchemaAttrs()
	attrs["workspace_id"] = schema.StringAttribute{
		Optional:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		Description:   "ID of the workspace this agent belongs to. Required when using WIF authentication. Not needed when using workspace_api_key.",
	}
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic agent. Supports WIF (workspace_id required) and workspace API key authentication. WIF takes precedence when both are configured.",
		Attributes:  attrs,
	}
}

func (r *WIFAgentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy plan — nothing to validate.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan WIFAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that at least one auth method will be available at apply time.
	// workspace_id may be unknown at plan time (e.g. referencing a not-yet-created workspace),
	// so WIF is considered configured based on the WIF credentials alone, not workspace_id.
	wifConfigured := r.data != nil && r.data.wif != nil
	apiKeyConfigured := r.data != nil && r.data.workspaceAPIKey != ""
	if r.data != nil && !wifConfigured && !apiKeyConfigured {
		resp.Diagnostics.AddError(
			"Missing credentials",
			"No authentication method is configured for anthropic_agent. "+
				"Set workspace_id together with WIF credentials (federation_rule_id, organization_id, service_account_id), "+
				"or set workspace_api_key in the provider block.",
		)
		return
	}

	// Skip version/updated_at unknowns on create (no prior state).
	if req.State.Raw.IsNull() {
		return
	}

	var state WIFAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !agentUserFieldsChanged(plan.AgentCoreModel, state.AgentCoreModel) {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("version"), types.Int64Unknown())...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("updated_at"), types.StringUnknown())...)
}

func (r *WIFAgentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// resolveCredentials returns the credentials for the agent API call.
// WIF is used when workspace_id is set and WIF is fully configured.
// workspace_api_key is used otherwise.
// When both are configured, WIF takes precedence.
func (r *WIFAgentResource) resolveCredentials(ctx context.Context, workspaceID string, diags interface{ AddError(string, string) }) auth.Credentials {
	if r.data == nil {
		diags.AddError("Provider not configured", "No provider data available.")
		return nil
	}

	if r.data.wif != nil && workspaceID != "" {
		tflog.Debug(ctx, "anthropic_agent: using WIF authentication", map[string]any{"workspace_id": workspaceID})
		return auth.WIFBearer{Config: r.data.wif, WorkspaceID: workspaceID}
	}

	if r.data.workspaceAPIKey != "" {
		tflog.Debug(ctx, "anthropic_agent: using workspace API key authentication")
		return auth.WorkspaceAPIKey{Key: r.data.workspaceAPIKey}
	}

	if workspaceID != "" && r.data.wifErr != nil {
		diags.AddError("Invalid WIF configuration", r.data.wifErr.Error())
	} else if workspaceID != "" {
		diags.AddError("Missing credentials",
			"workspace_id is set but WIF is not fully configured and workspace_api_key is not set. "+
				"Set federation_rule_id, organization_id, service_account_id in the provider block, "+
				"or set workspace_api_key.")
	} else {
		diags.AddError("Missing credentials",
			"No authentication method is configured for anthropic_agent. "+
				"Set workspace_api_key in the provider block, "+
				"or set workspace_id together with WIF credentials (federation_rule_id, organization_id, service_account_id).")
	}
	return nil
}

func (r *WIFAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WIFAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := r.resolveCredentials(ctx, data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := buildAgentBody(data.AgentCoreModel)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent configuration", err.Error())
		return
	}
	agent, err := client.NewAgentClient(creds).Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create agent: %s", err))
		return
	}
	data.AgentCoreModel.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WIFAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := r.resolveCredentials(ctx, data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	agent, err := client.NewAgentClient(creds).Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}
	if agent == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.AgentCoreModel.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WIFAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state WIFAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := r.resolveCredentials(ctx, data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := buildAgentBody(data.AgentCoreModel)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent configuration", err.Error())
		return
	}
	body["version"] = state.Version.ValueInt64()

	agent, err := client.NewAgentClient(creds).Update(ctx, data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update agent: %s", err))
		return
	}
	data.AgentCoreModel.fill(*agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WIFAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := r.resolveCredentials(ctx, data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.NewAgentClient(creds).Delete(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive agent: %s", err))
	}
}

// ImportState supports two formats:
//   - workspace_id/agent_id  (WIF path)
//   - agent_id               (workspace_api_key path — workspace_id left null)
func (r *WIFAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	switch len(parts) {
	case 2:
		if parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/agent_id or agent_id")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	case 1:
		if parts[0] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "agent_id must not be empty")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/agent_id or agent_id")
	}
}
