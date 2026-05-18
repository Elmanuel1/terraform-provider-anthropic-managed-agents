package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AgentCoreModel struct {
	Id          types.String `tfsdk:"id"`
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

func (m *AgentCoreModel) fill(a client.AgentResponse) {
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

func buildAgentBody(data AgentCoreModel) (map[string]any, error) {
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

func agentCoreSchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
	}
}

func agentUserFieldsChanged(plan, state AgentCoreModel) bool {
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
