package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VaultCredentialResource struct {
	data *providerData
}

type VaultCredentialModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	VaultId     types.String `tfsdk:"vault_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Auth        types.String `tfsdk:"auth"`
	Metadata    types.Map    `tfsdk:"metadata"`
	ForceDelete types.Bool   `tfsdk:"force_delete"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ArchivedAt  types.String `tfsdk:"archived_at"`
}

// mergeCredentialAuth merges the API response (which strips secrets) with the prior
// state auth JSON (which has secrets) so secrets are never lost across reads.
func mergeCredentialAuth(priorAuth types.String, apiAuth client.VaultCredentialAuthResponse) types.String {
	// Build the base map from non-secret API fields — used for both branches.
	base := map[string]any{"type": apiAuth.Type}
	if apiAuth.MCPServerURL != nil {
		base["mcp_server_url"] = *apiAuth.MCPServerURL
	}
	if apiAuth.ExpiresAt != nil {
		base["expires_at"] = *apiAuth.ExpiresAt
	}
	if apiAuth.Refresh != nil {
		refresh := map[string]any{}
		if apiAuth.Refresh.ClientID != "" {
			refresh["client_id"] = apiAuth.Refresh.ClientID
		}
		if apiAuth.Refresh.TokenEndpoint != "" {
			refresh["token_endpoint"] = apiAuth.Refresh.TokenEndpoint
		}
		if apiAuth.Refresh.TokenEndpointAuth != nil {
			refresh["token_endpoint_auth"] = map[string]any{"type": apiAuth.Refresh.TokenEndpointAuth.Type}
		}
		if apiAuth.Refresh.Resource != nil {
			refresh["resource"] = *apiAuth.Refresh.Resource
		}
		if apiAuth.Refresh.Scope != nil {
			refresh["scope"] = *apiAuth.Refresh.Scope
		}
		base["refresh"] = refresh
	}

	// No prior state (e.g. post-import) — return all non-secret API fields as-is.
	if priorAuth.IsNull() || priorAuth.IsUnknown() || priorAuth.ValueString() == "" {
		b, _ := json.Marshal(base)
		return types.StringValue(string(b))
	}

	// Prior state exists — overlay non-secret API fields on top to preserve secrets.
	var m map[string]any
	if err := json.Unmarshal([]byte(priorAuth.ValueString()), &m); err != nil {
		return priorAuth
	}
	for k, v := range base {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		return priorAuth
	}
	return types.StringValue(string(b))
}

func (m *VaultCredentialModel) fill(r client.VaultCredentialResponse, priorAuth types.String) {
	m.Id = types.StringValue(r.ID)
	m.VaultId = types.StringValue(r.VaultID)
	m.DisplayName = nullableString(r.DisplayName)
	m.CreatedAt = types.StringValue(r.CreatedAt)
	m.UpdatedAt = types.StringValue(r.UpdatedAt)
	m.ArchivedAt = nullableString(r.ArchivedAt)
	m.Metadata = fillMetadata(r.Metadata)
	m.Auth = mergeCredentialAuth(priorAuth, r.Auth)
}

func buildCredentialBody(data VaultCredentialModel) (map[string]any, error) {
	body := map[string]any{}
	if !data.DisplayName.IsNull() && !data.DisplayName.IsUnknown() {
		body["display_name"] = data.DisplayName.ValueString()
	}
	if !data.Auth.IsNull() && !data.Auth.IsUnknown() {
		var authObj map[string]any
		if err := json.Unmarshal([]byte(data.Auth.ValueString()), &authObj); err != nil {
			return nil, fmt.Errorf("invalid auth JSON: %w", err)
		}
		body["auth"] = authObj
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() && len(data.Metadata.Elements()) > 0 {
		meta := make(map[string]string)
		data.Metadata.ElementsAs(context.Background(), &meta, false)
		body["metadata"] = meta
	}
	return body, nil
}

func NewVaultCredentialResource() resource.Resource {
	return &VaultCredentialResource{}
}

var _ resource.Resource = &VaultCredentialResource{}
var _ resource.ResourceWithImportState = &VaultCredentialResource{}

func (r *VaultCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault_credential"
}

func (r *VaultCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a credential stored in an Anthropic vault.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workspace_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "ID of the workspace this resource belongs to.",
			},
			"vault_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "ID of the vault this credential belongs to.",
			},
			"display_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"auth": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Description: "JSON-encoded auth config. Secrets (token, access_token, refresh_token, client_secret) are write-only and never returned by the API — they are preserved from prior state on each read. Fields mcp_server_url, token_endpoint, and client_id are immutable after creation.",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Arbitrary string key-value pairs attached to the credential.",
			},
			"force_delete": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, permanently deletes the credential on destroy. When false (default), archives it instead.",
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

func (r *VaultCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VaultCredentialResource) requireWIF(diags interface{ AddError(string, string) }) bool {
	if r.data == nil || r.data.wif == nil {
		diags.AddError("Missing WIF configuration",
			"ANTHROPIC_FEDERATION_RULE_ID, ANTHROPIC_ORGANIZATION_ID, ANTHROPIC_SERVICE_ACCOUNT_ID, and one of TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC or TFC_WORKLOAD_IDENTITY_TOKEN are required for vault credential resources.")
		return false
	}
	return true
}

func (r *VaultCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VaultCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	priorAuth := data.Auth
	body, err := buildCredentialBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid credential configuration", err.Error())
		return
	}

	c := client.NewVaultCredentialClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	cred, err := c.Create(ctx, data.VaultId.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create vault credential: %s", err))
		return
	}
	data.fill(*cred, priorAuth)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VaultCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	priorAuth := data.Auth
	c := client.NewVaultCredentialClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	cred, err := c.Read(ctx, data.VaultId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault credential: %s", err))
		return
	}
	if cred == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	data.fill(*cred, priorAuth)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VaultCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	priorAuth := data.Auth
	body, err := buildCredentialBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid credential configuration", err.Error())
		return
	}

	c := client.NewVaultCredentialClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	cred, err := c.Update(ctx, data.VaultId.ValueString(), data.Id.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update vault credential: %s", err))
		return
	}
	data.fill(*cred, priorAuth)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VaultCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

	c := client.NewVaultCredentialClient(auth.WIFBearer{Config: r.data.wif, WorkspaceID: data.WorkspaceId.ValueString()})
	if data.ForceDelete.ValueBool() {
		if err := c.Delete(ctx, data.VaultId.ValueString(), data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete vault credential: %s", err))
		}
	} else {
		if err := c.Archive(ctx, data.VaultId.ValueString(), data.Id.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive vault credential: %s", err))
		}
	}
}

func (r *VaultCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/vault_id/credential_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vault_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}
