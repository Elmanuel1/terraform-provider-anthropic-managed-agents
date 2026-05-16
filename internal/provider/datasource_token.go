package provider

import (
	"context"
	"fmt"

	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic-wif/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type tokenDataSource struct {
	data *providerData
}

type tokenDataSourceModel struct {
	WorkspaceName types.String `tfsdk:"workspace_name"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	TokenPrefix   types.String `tfsdk:"token_prefix"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
}

func (d *tokenDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "anthropic-wif_token"
}

func (d *tokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Mints a WIF token for the given workspace and returns metadata (not the token itself).",
		Attributes: map[string]schema.Attribute{
			"workspace_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the Anthropic workspace as it appears in the Console.",
			},
			"workspace_id": schema.StringAttribute{
				Computed:    true,
				Description: "Resolved workspace ID.",
			},
			"token_prefix": schema.StringAttribute{
				Computed:    true,
				Description: "First 20 characters of the minted token — confirms exchange succeeded without exposing the full token.",
			},
			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "RFC3339 timestamp when the token expires.",
			},
		},
	}
}

func (d *tokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.data = data
}

func (d *tokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model tokenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceName := model.WorkspaceName.ValueString()

	wc := client.NewWorkspaceClient(d.data.client)
	workspaceID, err := wc.ResolveByName(ctx, workspaceName)
	if err != nil {
		resp.Diagnostics.AddError("Workspace resolution failed", err.Error())
		return
	}

	auth.LogJWTClaims(ctx, d.data.client.WIF)

	tflog.Info(ctx, "minting WIF token", map[string]any{
		"workspace_name": workspaceName,
		"workspace_id":   workspaceID,
	})

	token, err := auth.MintToken(ctx, d.data.client.WIF, workspaceID)
	if err != nil {
		resp.Diagnostics.AddError("Token minting failed", err.Error())
		return
	}

	prefix := token.AccessToken
	if len(prefix) > 20 {
		prefix = prefix[:20] + "..."
	}

	tflog.Info(ctx, "WIF token minted", map[string]any{
		"workspace_id": workspaceID,
		"token_prefix": prefix,
		"expires_at":   token.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})

	model.WorkspaceID = types.StringValue(workspaceID)
	model.TokenPrefix = types.StringValue(prefix)
	model.ExpiresAt = types.StringValue(token.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
