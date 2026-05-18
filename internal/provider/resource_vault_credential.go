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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type WIFVaultCredentialResource struct {
	data *providerData
}

type WIFVaultCredentialModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	VaultId     types.String `tfsdk:"vault_id"`
	DisplayName types.String `tfsdk:"display_name"`
	ForceDelete types.Bool   `tfsdk:"force_delete"`
	// Non-secret auth fields
	AuthType              types.String `tfsdk:"auth_type"`
	MCPServerURL          types.String `tfsdk:"mcp_server_url"`
	ExpiresAt             types.String `tfsdk:"expires_at"`
	TokenEndpoint         types.String `tfsdk:"token_endpoint"`
	ClientID              types.String `tfsdk:"client_id"`
	TokenEndpointAuthType types.String `tfsdk:"token_endpoint_auth_type"`
	Scope                 types.String `tfsdk:"scope"`
	Resource              types.String `tfsdk:"resource"`
	// Write-only secret fields
	Token        types.String `tfsdk:"token"`
	AccessToken  types.String `tfsdk:"access_token"`
	RefreshToken types.String `tfsdk:"refresh_token"`
	ClientSecret types.String `tfsdk:"client_secret"`
	// Server-managed
	Metadata   types.Map    `tfsdk:"metadata"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
	ArchivedAt types.String `tfsdk:"archived_at"`
}

func (m *WIFVaultCredentialModel) fill(r client.VaultCredentialResponse) {
	m.Id = types.StringValue(r.ID)
	m.VaultId = types.StringValue(r.VaultID)
	m.DisplayName = nullableString(r.DisplayName)
	m.AuthType = types.StringValue(r.Auth.Type)
	if r.Auth.MCPServerURL != nil {
		m.MCPServerURL = types.StringValue(strings.TrimRight(*r.Auth.MCPServerURL, "/"))
	}
	if r.Auth.ExpiresAt != nil {
		m.ExpiresAt = types.StringValue(*r.Auth.ExpiresAt)
	} else {
		m.ExpiresAt = types.StringNull()
	}
	if r.Auth.Refresh != nil {
		m.TokenEndpoint = types.StringValue(r.Auth.Refresh.TokenEndpoint)
		m.ClientID = types.StringValue(r.Auth.Refresh.ClientID)
		if r.Auth.Refresh.TokenEndpointAuth != nil {
			m.TokenEndpointAuthType = types.StringValue(r.Auth.Refresh.TokenEndpointAuth.Type)
		}
		m.Scope = nullableString(r.Auth.Refresh.Scope)
		m.Resource = nullableString(r.Auth.Refresh.Resource)
	} else {
		m.TokenEndpoint = types.StringNull()
		m.ClientID = types.StringNull()
		m.TokenEndpointAuthType = types.StringNull()
		m.Scope = types.StringNull()
		m.Resource = types.StringNull()
	}
	m.CreatedAt = types.StringValue(r.CreatedAt)
	m.UpdatedAt = types.StringValue(r.UpdatedAt)
	m.ArchivedAt = nullableString(r.ArchivedAt)
	m.Metadata = fillMetadata(r.Metadata)
	// Write-only fields (token, access_token, refresh_token, client_secret) are NOT set here.
	// They are never returned by the API and must not be written to state.
}

func buildCredentialBody(data WIFVaultCredentialModel) (map[string]any, error) {
	authObj := map[string]any{
		"type":           data.AuthType.ValueString(),
		"mcp_server_url": data.MCPServerURL.ValueString(),
	}

	switch data.AuthType.ValueString() {
	case "static_bearer":
		if !data.Token.IsNull() && !data.Token.IsUnknown() {
			authObj["token"] = data.Token.ValueString()
		}
	case "mcp_oauth":
		if !data.AccessToken.IsNull() && !data.AccessToken.IsUnknown() {
			authObj["access_token"] = data.AccessToken.ValueString()
		}
		if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
			authObj["expires_at"] = data.ExpiresAt.ValueString()
		}
		if !data.TokenEndpoint.IsNull() && !data.TokenEndpoint.IsUnknown() {
			refresh := map[string]any{
				"token_endpoint": data.TokenEndpoint.ValueString(),
				"client_id":      data.ClientID.ValueString(),
			}
			if !data.RefreshToken.IsNull() && !data.RefreshToken.IsUnknown() {
				refresh["refresh_token"] = data.RefreshToken.ValueString()
			}
			if !data.TokenEndpointAuthType.IsNull() && !data.TokenEndpointAuthType.IsUnknown() {
				tea := map[string]any{"type": data.TokenEndpointAuthType.ValueString()}
				if !data.ClientSecret.IsNull() && !data.ClientSecret.IsUnknown() {
					tea["client_secret"] = data.ClientSecret.ValueString()
				}
				refresh["token_endpoint_auth"] = tea
			}
			if !data.Scope.IsNull() && !data.Scope.IsUnknown() {
				refresh["scope"] = data.Scope.ValueString()
			}
			if !data.Resource.IsNull() && !data.Resource.IsUnknown() {
				refresh["resource"] = data.Resource.ValueString()
			}
			authObj["refresh"] = refresh
		}
	default:
		return nil, fmt.Errorf("unsupported auth_type %q: must be static_bearer or mcp_oauth", data.AuthType.ValueString())
	}

	body := map[string]any{"auth": authObj}
	if !data.DisplayName.IsNull() && !data.DisplayName.IsUnknown() {
		body["display_name"] = data.DisplayName.ValueString()
	}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		meta := make(map[string]string)
		data.Metadata.ElementsAs(context.Background(), &meta, false)
		body["metadata"] = meta
	}
	return body, nil
}

func NewWIFVaultCredentialResource() resource.Resource {
	return &WIFVaultCredentialResource{}
}

var _ resource.Resource = &WIFVaultCredentialResource{}
var _ resource.ResourceWithImportState = &WIFVaultCredentialResource{}

func (r *WIFVaultCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault_credential"
}

func (r *WIFVaultCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"force_delete": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, permanently deletes the credential on destroy. When false (default), archives it instead.",
			},
			"auth_type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   `Credential type: "static_bearer" or "mcp_oauth". Changing this forces a new resource.`,
			},
			"mcp_server_url": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "MCP server URL. Immutable after creation.",
			},
			"expires_at": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OAuth token expiry timestamp, returned by the API.",
			},
			"token_endpoint": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "OAuth token endpoint URL. Immutable after creation.",
			},
			"client_id": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "OAuth client ID. Immutable after creation.",
			},
			"token_endpoint_auth_type": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   `OAuth token endpoint auth method: "none", "client_secret_basic", or "client_secret_post". Changing this forces a new resource.`,
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OAuth scope.",
			},
			"resource": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OAuth resource indicator.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "Static bearer token. Write-only — never stored in state.",
			},
			"access_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "OAuth access token. Write-only — never stored in state.",
			},
			"refresh_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "OAuth refresh token. Write-only — never stored in state.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "OAuth client secret. Write-only — never stored in state.",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Arbitrary string key-value pairs attached to the credential.",
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

func (r *WIFVaultCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WIFVaultCredentialResource) requireWIF(diags interface{ AddError(string, string) }) bool {
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

func (r *WIFVaultCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WIFVaultCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// WriteOnly fields are absent from the plan's new state — read them from config.
	var cfg WIFVaultCredentialModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Token = cfg.Token
	data.AccessToken = cfg.AccessToken
	data.RefreshToken = cfg.RefreshToken
	data.ClientSecret = cfg.ClientSecret

	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

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
	data.fill(*cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFVaultCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WIFVaultCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

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
	data.fill(*cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFVaultCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WIFVaultCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// WriteOnly fields are absent from the plan's new state — read them from config.
	var cfg WIFVaultCredentialModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Token = cfg.Token
	data.AccessToken = cfg.AccessToken
	data.RefreshToken = cfg.RefreshToken
	data.ClientSecret = cfg.ClientSecret

	if !r.requireWIF(&resp.Diagnostics) {
		return
	}

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
	data.fill(*cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WIFVaultCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WIFVaultCredentialModel
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

func (r *WIFVaultCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: workspace_id/vault_id/credential_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vault_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}
