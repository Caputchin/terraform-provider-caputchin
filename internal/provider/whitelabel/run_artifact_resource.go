// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

var (
	_ resource.Resource                     = (*runArtifactResource)(nil)
	_ resource.ResourceWithConfigure        = (*runArtifactResource)(nil)
	_ resource.ResourceWithConfigValidators = (*runArtifactResource)(nil)
	_ resource.ResourceWithImportState      = (*runArtifactResource)(nil)
)

// NewRunArtifactResource is the factory consumed by the provider's Resources()
// list. The resource uploads the headless replay artifact (run.js + optional
// wasm/js modules) that lets a custom game be used as a verification gate.
// The playable bundle stays on the customer's CDN (the widget's `game-src`
// attribute); only the deterministic replay artifact lives on the platform.
func NewRunArtifactResource() resource.Resource {
	return &runArtifactResource{}
}

type runArtifactResource struct {
	client *client.Client
}

func (r *runArtifactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_game_run_artifact"
}

func (r *runArtifactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Uploads the headless replay artifact for a custom game (run.js plus optional wasm/js modules) so the game becomes eligible as a verification gate. The custom game must already exist via `caputchin_customized_game` with `source = \"custom\"`. Set exactly one of `troop_id` / `site_id`. Free for all tiers.\n\nDrift detection: the resource never reads local files at plan time, so a file edit alone won't trigger an update. Either taint the resource or pass a `source_hash` that the HCL recomputes via `sha256(filesha256(var.run_path), [for m in var.module_paths : filesha256(m)])` so a content change drives a new plan.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Synthetic resource id encoding the composite key as `scope|id|game` (scope is troop|site). Matches the import id.",
				Computed:      true,
				PlanModifiers: useStateForUnknown(),
			},
			"troop_id": schema.StringAttribute{
				Description:   "Troop id. Exactly one of `troop_id` / `site_id` is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"site_id": schema.StringAttribute{
				Description:   "Site id. Exactly one of `troop_id` / `site_id` is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"game_id": schema.StringAttribute{
				Description:   "Custom game id, e.g. `customer/my-game`. Forces replacement.",
				Required:      true,
				PlanModifiers: requiresReplace(),
			},
			"run_path": schema.StringAttribute{
				Description: "Local file path to the headless `run.js` entry. The file's basename is ignored; the server stores it as `run.js`.",
				Required:    true,
			},
			"module_paths": schema.ListAttribute{
				Description: "Local file paths for additional modules the run entry imports (`.wasm` or `.js`). Each filename must match `^[a-zA-Z0-9_-]+\\.(wasm|js)$` and is forwarded as-is. Max 16 modules, 5 MB per file, 10 MB combined.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"source_hash": schema.StringAttribute{
				Description: "Optional caller-computed drift signal. When this string changes between plans, the resource re-uploads. Recommended: `sha256(filesha256(var.run_path), [for m in var.module_paths : filesha256(m)])`.",
				Optional:    true,
			},
			"version_hash": schema.StringAttribute{
				Description: "Server-assigned content-hash of the active artifact. Read-only.",
				Computed:    true,
			},
			"self_check_ok": schema.BoolAttribute{
				Description: "Replay self-check verdict at upload time. Only `true` makes the install gate-eligible. Read-only.",
				Computed:    true,
			},
			"uploaded_at": schema.StringAttribute{
				Description: "ISO timestamp the active artifact was first vendored. Read-only.",
				Computed:    true,
			},
		},
	}
}

func (r *runArtifactResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("troop_id"), path.MatchRoot("site_id")),
	}
}

func (r *runArtifactResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected-provider-data", fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *runArtifactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan runArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upload(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *runArtifactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state runArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var detail runArtifactDetailWire
	err := r.client.Get(ctx, runArtifactDetailPath(state.TroopID, state.SiteID, state.GameID), &detail)
	if client.IsNotFound(err) {
		// No artifact uploaded (or it was removed out-of-band). Drop from state
		// so the next plan re-uploads.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("run-artifact-read-failed", err.Error())
		return
	}
	state.VersionHash = types.StringValue(detail.VersionHash)
	state.SelfCheckOK = types.BoolValue(detail.SelfCheckOK)
	state.UploadedAt = types.StringValue(detail.UploadedAt)
	state.ID = types.StringValue(buildRunArtifactID(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *runArtifactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan runArtifactModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upload(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *runArtifactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state runArtifactModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.Delete(ctx, runArtifactPath(state.TroopID, state.SiteID, state.GameID))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("run-artifact-delete-failed", err.Error())
	}
}

// Import id format: scope|id|game.
func (r *runArtifactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "|", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("invalid-import-id", "expected scope|id|game (scope is troop|site)")
		return
	}
	scope, id, game := parts[0], parts[1], parts[2]
	switch scope {
	case "troop":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("troop_id"), id)...)
	case "site":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), id)...)
	default:
		resp.Diagnostics.AddError("invalid-import-id", "scope must be 'troop' or 'site'")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("game_id"), game)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *runArtifactResource) upload(ctx context.Context, m *runArtifactModel, diags *diag.Diagnostics) {
	runBytes, err := os.ReadFile(m.RunPath.ValueString())
	if err != nil {
		diags.AddError("run-artifact-read-run-failed", fmt.Sprintf("Couldn't read run_path %q: %s", m.RunPath.ValueString(), err))
		return
	}
	parts := []client.MultipartFile{
		{FieldName: "run", Filename: "run.js", Bytes: runBytes},
	}

	if !m.ModulePaths.IsNull() && !m.ModulePaths.IsUnknown() {
		var paths []string
		diags.Append(m.ModulePaths.ElementsAs(ctx, &paths, false)...)
		if diags.HasError() {
			return
		}
		for _, p := range paths {
			modBytes, err := os.ReadFile(p)
			if err != nil {
				diags.AddError("run-artifact-read-module-failed", fmt.Sprintf("Couldn't read module %q: %s", p, err))
				return
			}
			parts = append(parts, client.MultipartFile{
				FieldName: "module",
				Filename:  filepath.Base(p),
				Bytes:     modBytes,
			})
		}
	}

	var out runArtifactUploadResponseWire
	if err := r.client.PutMultipart(ctx, runArtifactPath(m.TroopID, m.SiteID, m.GameID), parts, &out); err != nil {
		diags.AddError("run-artifact-upload-failed", err.Error())
		return
	}
	m.VersionHash = types.StringValue(out.VersionHash)
	m.SelfCheckOK = types.BoolValue(out.SelfCheckOK)
	// uploaded_at isn't in the upload response; mark unknown and let the next
	// Read fill it in. The framework will resolve "unknown" against state on
	// the subsequent plan; until then it shows as a computed value pending
	// refresh.
	m.UploadedAt = types.StringUnknown()
	m.ID = types.StringValue(buildRunArtifactID(*m))
}
