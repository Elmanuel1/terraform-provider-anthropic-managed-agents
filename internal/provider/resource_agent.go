package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AgentResource struct {
	data *providerData
}

type AgentModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Model       types.String `tfsdk:"model"`
	System      types.String `tfsdk:"system"`
	Description types.String `tfsdk:"description"`
	Version     types.Int64  `tfsdk:"version"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

type agentAPIResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model struct {
		ID string `json:"id"`
	} `json:"model"`
	System      *string `json:"system"`
	Description *string `json:"description"`
	Version     int     `json:"version"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	ArchivedAt  *string `json:"archived_at"`
}

func (m *AgentModel) fill(a agentAPIResponse) {
	m.Id = types.StringValue(a.ID)
	m.Name = types.StringValue(a.Name)
	m.Model = types.StringValue(a.Model.ID)
	m.Version = types.Int64Value(int64(a.Version))
	m.CreatedAt = types.StringValue(a.CreatedAt)
	m.UpdatedAt = types.StringValue(a.UpdatedAt)
	m.System = nullableString(a.System)
	m.Description = nullableString(a.Description)
	m.ArchivedAt = nullableString(a.ArchivedAt)
}

func NewAgentResource() resource.Resource {
	return &AgentResource{}
}

var _ resource.Resource = &AgentResource{}
var _ resource.ResourceWithImportState = &AgentResource{}

func (r *AgentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (r *AgentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic agent.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true},
			"model": schema.StringAttribute{
				Required:    true,
				Description: "Model ID, e.g. claude-opus-4-7.",
			},
			"system":      schema.StringAttribute{Optional: true, Computed: true},
			"description": schema.StringAttribute{Optional: true, Computed: true},
			"version": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
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

func (r *AgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":  data.Name.ValueString(),
		"model": map[string]string{"id": data.Model.ValueString()},
	}
	if !data.System.IsNull() && !data.System.IsUnknown() {
		body["system"] = data.System.ValueString()
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}

	raw, status, err := doRequest(ctx, r.data, http.MethodPost, "/v1/agents", body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create agent: %s", err))
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create agent, status %d: %s", status, raw))
		return
	}

	var a agentAPIResponse
	if err := json.Unmarshal(raw, &a); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse agent response: %s", err))
		return
	}
	data.fill(a)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, status, err := doRequest(ctx, r.data, http.MethodGet, "/v1/agents/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent: %s", err))
		return
	}
	if status == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent, status %d: %s", status, raw))
		return
	}

	var a agentAPIResponse
	if err := json.Unmarshal(raw, &a); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse agent response: %s", err))
		return
	}
	data.fill(a)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":  data.Name.ValueString(),
		"model": map[string]string{"id": data.Model.ValueString()},
	}
	if !data.System.IsNull() && !data.System.IsUnknown() {
		body["system"] = data.System.ValueString()
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}

	raw, status, err := doRequest(ctx, r.data, http.MethodPost, "/v1/agents/"+data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update agent: %s", err))
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update agent, status %d: %s", status, raw))
		return
	}

	var a agentAPIResponse
	if err := json.Unmarshal(raw, &a); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse agent response: %s", err))
		return
	}
	data.fill(a)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, status, err := doRequest(ctx, r.data, http.MethodPost, "/v1/agents/"+data.Id.ValueString()+"/archive", nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive agent: %s", err))
		return
	}
	if status != http.StatusOK && status != http.StatusNotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive agent, status %d", status))
	}
}

func (r *AgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
