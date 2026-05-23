package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
	"github.com/Elmanuel1/terraform-provider-anthropic/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SkillResource struct {
	data *providerData
}

type SkillModel struct {
	ID           types.String `tfsdk:"id"`
	DisplayTitle types.String `tfsdk:"display_title"`
	Description  types.String `tfsdk:"description"`
	SourceDir    types.String `tfsdk:"source_dir"`
	SourceHash   types.String `tfsdk:"source_hash"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (m *SkillModel) fill(s client.SkillResponse) {
	m.ID = types.StringValue(s.ID)
	m.DisplayTitle = types.StringValue(s.DisplayTitle)
	m.Description = nullableString(s.Description)
	m.CreatedAt = types.StringValue(s.CreatedAt)
	m.UpdatedAt = types.StringValue(s.UpdatedAt)
	// SourceDir and SourceHash are not set here — they are managed locally.
}

func NewSkillResource() resource.Resource {
	return &SkillResource{}
}

var _ resource.Resource = &SkillResource{}
var _ resource.ResourceWithImportState = &SkillResource{}

func (r *SkillResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

func (r *SkillResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anthropic skill. Skills are uploaded from a local directory containing a SKILL.md file.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"display_title": schema.StringAttribute{
				Required:      true,
				Description:   "Human-readable title for the skill.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Optional description of the skill.",
			},
			"source_dir": schema.StringAttribute{
				Required:    true,
				Description: "Local directory path containing the skill source files. Must contain a SKILL.md at the root.",
			},
			"source_hash": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 hash of the skill source directory contents. Changes when source files change, triggering a new skill version.",
				PlanModifiers: []planmodifier.String{
					sourceHashPlanModifier{},
				},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *SkillResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SkillResource) requireAdminKey(diags interface{ AddError(string, string) }) bool {
	if r.data == nil || r.data.adminKey == "" {
		diags.AddError("Missing admin API key",
			"Set admin_api_key in the provider block or ANTHROPIC_ADMIN_API_KEY environment variable. Required for anthropic_skill.")
		return false
	}
	return true
}

func (r *SkillResource) skillClient() *client.SkillClient {
	return client.NewSkillClient(auth.AdminAPIKey{Key: r.data.adminKey, Beta: auth.SkillsBeta})
}

func (r *SkillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var data SkillModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceDir := data.SourceDir.ValueString()
	hash, err := computeSourceHash(sourceDir)
	if err != nil {
		resp.Diagnostics.AddError("Source Hash Error", fmt.Sprintf("Unable to compute source hash: %s", err))
		return
	}

	var desc *string
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		v := data.Description.ValueString()
		desc = &v
	}

	c := r.skillClient()
	s, err := c.Create(ctx, data.DisplayTitle.ValueString(), desc, sourceDir)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create skill: %s", err))
		return
	}

	data.fill(*s)
	data.SourceDir = types.StringValue(sourceDir)
	data.SourceHash = types.StringValue(hash)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SkillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var data SkillModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := r.skillClient()
	s, err := c.Read(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read skill: %s", err))
		return
	}
	if s == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	sourceDir := data.SourceDir
	sourceHash := data.SourceHash

	data.fill(*s)
	data.SourceDir = sourceDir
	data.SourceHash = sourceHash
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SkillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var plan, state SkillModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := r.skillClient()

	if !plan.SourceHash.Equal(state.SourceHash) {
		_, err := c.CreateVersion(ctx, state.ID.ValueString(), plan.SourceDir.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create skill version: %s", err))
			return
		}
	}

	// Refresh state after update.
	s, err := c.Read(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read skill after update: %s", err))
		return
	}
	if s == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hash, err := computeSourceHash(plan.SourceDir.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Source Hash Error", fmt.Sprintf("Unable to compute source hash: %s", err))
		return
	}

	plan.fill(*s)
	plan.SourceDir = types.StringValue(plan.SourceDir.ValueString())
	plan.SourceHash = types.StringValue(hash)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SkillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.requireAdminKey(&resp.Diagnostics) {
		return
	}
	var data SkillModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c := r.skillClient()
	if err := c.Delete(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete skill: %s", err))
	}
}

func (r *SkillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// computeSourceHash returns a stable SHA-256 hash of all files in sourceDir,
// sorted by relative path. File paths and contents are both included in the hash.
func computeSourceHash(sourceDir string) (string, error) {
	var files []string
	err := filepath.Walk(sourceDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(sourceDir, p)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking %q: %w", sourceDir, err)
	}
	sort.Strings(files)

	h := sha256.New()
	for _, rel := range files {
		content, err := os.ReadFile(filepath.Join(sourceDir, rel))
		if err != nil {
			return "", fmt.Errorf("reading %q: %w", rel, err)
		}
		fmt.Fprintf(h, "%s\n", rel)
		h.Write(content)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// sourceHashPlanModifier recomputes source_hash from source_dir on every plan.
type sourceHashPlanModifier struct{}

func (m sourceHashPlanModifier) Description(_ context.Context) string {
	return "Recomputes source_hash from source_dir on every plan."
}

func (m sourceHashPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Recomputes `source_hash` from `source_dir` on every plan."
}

func (m sourceHashPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var sourceDir types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("source_dir"), &sourceDir)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if sourceDir.IsNull() || sourceDir.IsUnknown() {
		resp.PlanValue = types.StringUnknown()
		return
	}

	hash, err := computeSourceHash(sourceDir.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Source Hash Error",
			fmt.Sprintf("Unable to compute source hash from %q: %s", sourceDir.ValueString(), err))
		return
	}
	resp.PlanValue = types.StringValue(hash)
}
