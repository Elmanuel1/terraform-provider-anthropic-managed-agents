package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/client"
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
	agentCoreModel
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
		Description:   "ID of the workspace this agent belongs to. Required when using WIF authentication. Not needed when using api_key.",
	}
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic agent. Supports WIF (workspace_id required) and workspace API key authentication. WIF takes precedence when both are configured.",
		Attributes:  attrs,
	}
}

func (r *WIFAgentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var plan, state WIFAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !agentUserFieldsChanged(plan.agentCoreModel, state.agentCoreModel) {
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

// resolveCredentials returns the credentials to use for the agent API call.
// WIF takes precedence when workspace_id is set and WIF is fully configured.
// Falls back to the workspace API key when WIF is not available.
func (r *WIFAgentResource) resolveCredentials(ctx context.Context, workspaceID string, diags interface{ AddError(string, string) }) auth.Credentials {
	if r.data == nil {
		diags.AddError("Provider not configured", "No provider data available.")
		return nil
	}

	wifReady := r.data.wif != nil && workspaceID != ""

	if wifReady {
		if r.data.wif != nil && r.data.wifErr == nil {
			tflog.Debug(ctx, "anthropic_agent: using WIF authentication", map[string]any{"workspace_id": workspaceID})
			return auth.WIFBearer{Config: r.data.wif, WorkspaceID: workspaceID}
		}
	}

	if r.data.apiKey != "" {
		if wifReady {
			tflog.Warn(ctx, "anthropic_agent: WIF config incomplete, falling back to api_key", map[string]any{"workspace_id": workspaceID})
		} else {
			tflog.Debug(ctx, "anthropic_agent: using workspace API key authentication")
		}
		return auth.WorkspaceAPIKey{Key: r.data.apiKey}
	}

	// Neither auth method is available — report a clear error.
	if workspaceID != "" && r.data.wifErr != nil {
		diags.AddError("Invalid WIF configuration", r.data.wifErr.Error())
	} else if workspaceID != "" {
		diags.AddError("Missing credentials",
			"workspace_id is set but WIF is not configured and no api_key is available. "+
				"Set federation_rule_id, organization_id, service_account_id (or ANTHROPIC_FEDERATION_RULE_ID, ANTHROPIC_ORGANIZATION_ID, ANTHROPIC_SERVICE_ACCOUNT_ID) "+
				"or provide api_key (or ANTHROPIC_API_KEY).")
	} else {
		diags.AddError("Missing credentials",
			"No authentication method is configured for anthropic_agent. "+
				"Provide api_key (or ANTHROPIC_API_KEY), or set workspace_id together with WIF credentials.")
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

	body, err := buildAgentBody(data.agentCoreModel)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent configuration", err.Error())
		return
	}
	agent, err := client.NewAgentClient(creds).Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create agent: %s", err))
		return
	}
	data.agentCoreModel.fill(*agent)
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
	data.agentCoreModel.fill(*agent)
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

	body, err := buildAgentBody(data.agentCoreModel)
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
	data.agentCoreModel.fill(*agent)
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
//   - agent_id               (API key path — workspace_id left null)
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
