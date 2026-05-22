package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AgentDataSource struct {
	data *providerData
}

// AgentDataModel embeds AgentCoreModel exactly as WIFAgentModel does.
// The framework resolves embedded tfsdk fields correctly, and fill() is
// inherited — no manual field copy needed.
type AgentDataModel struct {
	WorkspaceId types.String `tfsdk:"workspace_id"`
	AgentCoreModel
}

// agentCoreDSAttrs returns datasource/schema equivalents of agentCoreSchemaAttrs().
// Plan modifiers and defaults are not applicable to data sources; id is Required
// rather than Computed. Keep descriptions in sync with agentCoreSchemaAttrs().
func agentCoreDSAttrs() map[string]dsschema.Attribute {
	return map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Required:    true,
			Description: "Agent ID (agt_...).",
		},
		"name":  dsschema.StringAttribute{Computed: true},
		"model": dsschema.StringAttribute{Computed: true, Description: "Model ID, e.g. claude-opus-4-7 or claude-sonnet-4-6."},
		"model_speed": dsschema.StringAttribute{
			Computed:    true,
			Description: "Inference speed: standard (default) or fast.",
		},
		"system":      dsschema.StringAttribute{Computed: true},
		"description": dsschema.StringAttribute{Computed: true},
		"tools": dsschema.StringAttribute{
			Computed:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: `JSON-encoded tools array. Example: [{"type":"agent_toolset_20260401"}]`,
		},
		"mcp_servers": dsschema.StringAttribute{
			Computed:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: `JSON-encoded MCP servers array. Example: [{"name":"my-server","type":"url","url":"https://..."}]. Maximum 20, names must be unique.`,
		},
		"skills": dsschema.StringAttribute{
			Computed:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: `JSON-encoded skills array. Example: [{"type":"anthropic","skill_id":"xlsx"}]. Maximum 20.`,
		},
		"multiagent": dsschema.StringAttribute{
			Computed:    true,
			CustomType:  jsontypes.NormalizedType{},
			Description: `JSON-encoded multiagent coordinator config. Example: {"type":"coordinator","agents":["agent_id_1","agent_id_2"]}.`,
		},
		"metadata": dsschema.MapAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Arbitrary string key-value pairs attached to the agent.",
		},
		"version":     dsschema.Int64Attribute{Computed: true},
		"created_at":  dsschema.StringAttribute{Computed: true},
		"updated_at":  dsschema.StringAttribute{Computed: true},
		"archived_at": dsschema.StringAttribute{Computed: true},
	}
}

func NewAgentDataSource() datasource.DataSource {
	return &AgentDataSource{}
}

var _ datasource.DataSource = &AgentDataSource{}
var _ datasource.DataSourceWithConfigure = &AgentDataSource{}

func (d *AgentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (d *AgentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := agentCoreDSAttrs()
	attrs["workspace_id"] = dsschema.StringAttribute{
		Optional:    true,
		Description: "ID of the workspace this agent belongs to. Required when using WIF authentication. Not needed when using workspace_api_key.",
	}
	resp.Schema = dsschema.Schema{
		Description: "Reads an existing Anthropic agent by ID.",
		Attributes:  attrs,
	}
}

func (d *AgentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AgentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds := resolveWorkspaceCredentials(ctx, d.data, "data.anthropic_agent", data.WorkspaceId.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	a, err := client.NewAgentClient(creds).Read(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}
	if a == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Agent %q not found.", data.Id.ValueString()))
		return
	}

	if err := data.AgentCoreModel.fill(*a); err != nil {
		resp.Diagnostics.AddError("Internal Error", fmt.Sprintf("marshaling agent response: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
