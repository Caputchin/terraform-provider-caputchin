// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

var (
	_ resource.Resource                     = (*presetResource)(nil)
	_ resource.ResourceWithConfigure        = (*presetResource)(nil)
	_ resource.ResourceWithConfigValidators = (*presetResource)(nil)
	_ resource.ResourceWithImportState      = (*presetResource)(nil)
)

// NewPresetResource is the factory consumed by the provider's Resources() list.
func NewPresetResource() resource.Resource {
	return &presetResource{}
}

type presetResource struct {
	client *client.Client
}

func requiresReplace() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

// useStateForUnknown keeps the computed synthetic `id` stable across updates
// (it only changes on replace, since every identity attr is RequiresReplace),
// avoiding a spurious "known after apply" on the id during plain value updates.
func useStateForUnknown() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
}

func (r *presetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_white_label_preset"
}

func (r *presetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A single white-label / game-customization preset. Per-preset granularity: one resource is one preset row. Set exactly one of `troop_id` (troop-wide baseline) or `site_id` (per-site override). Omit `game_id` for a widget-shell preset (Apex tier); set it for a game-axis preset (configuration = Solo+, skin/locale = Alpha+). A game-axis preset requires the game registered first: declare a `caputchin_customized_game` for the same scope + game and set `game_id = caputchin_customized_game.<name>.game_id` so it applies first, else the API rejects the write with `game-not-registered`. Changing scope, game, axis, or name forces replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Synthetic resource id encoding the composite key as `scope|id|game|axis|name` (scope is troop|site; game empty for a widget-shell preset). Matches the import id.",
				Computed:      true,
				PlanModifiers: useStateForUnknown(),
			},
			"troop_id": schema.StringAttribute{
				Description:   "Troop id for a troop-wide preset. Exactly one of troop_id / site_id is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"site_id": schema.StringAttribute{
				Description:   "Site id for a per-site preset. Exactly one of troop_id / site_id is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"game_id": schema.StringAttribute{
				Description:   "Game id (e.g. `caputchin/games/leaf`) for a game-axis preset. Omit for a widget-shell (white-label) preset. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"axis": schema.StringAttribute{
				Description: "Override axis: `locale`, `skin`, or `configuration`. Forces replacement.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("locale", "skin", "configuration"),
				},
				PlanModifiers: requiresReplace(),
			},
			"name": schema.StringAttribute{
				Description:   "Preset name. Forces replacement (rename = destroy + recreate).",
				Required:      true,
				PlanModifiers: requiresReplace(),
			},
			"values": schema.StringAttribute{
				Description:   "JSON object of preset leaf values (use `jsonencode({...})`). Empty-string / null leaves are dropped to match what the API persists (no perpetual diff), so they neither apply nor drift. Compared semantically, so key order and whitespace never produce a diff.",
				Required:      true,
				CustomType:    jsontypes.NormalizedType{},
				PlanModifiers: []planmodifier.String{dropEmptyLeavesModifier()},
			},
			"updated_at": schema.StringAttribute{
				Description: "Server timestamp (ISO 8601) of the last write.",
				Computed:    true,
			},
		},
	}
}

func (r *presetResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("troop_id"), path.MatchRoot("site_id")),
	}
}

func (r *presetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *presetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan presetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.write(ctx, &plan, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *presetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state presetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var env presetEnvelope
	err := r.client.Get(ctx, presetPath(state.TroopID, state.SiteID, state.GameID, state.Axis.ValueString(), state.Name.ValueString()), &env)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("white-label-preset-read-failed", err.Error())
		return
	}
	if err := applyPresetWire(&state, env.Preset); err != nil {
		resp.Diagnostics.AddError("white-label-preset-decode-failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *presetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan presetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.write(ctx, &plan, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *presetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state presetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.Delete(ctx, presetPath(state.TroopID, state.SiteID, state.GameID, state.Axis.ValueString(), state.Name.ValueString()))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("white-label-preset-delete-failed", err.Error())
	}
}

// Import id format: scope|id|game|axis|name. `game` is empty for a
// widget-shell preset. The game segment may itself contain slashes, and the
// pipe delimiter keeps it intact (game ids look like caputchin/games/leaf).
//
//	troop|troop_123||skin|midnight
//	site|site_123|caputchin/games/leaf|configuration|hard
func (r *presetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "|", 5)
	if len(parts) != 5 {
		resp.Diagnostics.AddError("invalid-import-id", "expected scope|id|game|axis|name (scope is troop|site; game is empty for a widget-shell preset)")
		return
	}
	scope, id, game, axis, name := parts[0], parts[1], parts[2], parts[3], parts[4]
	switch scope {
	case "troop":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("troop_id"), id)...)
	case "site":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), id)...)
	default:
		resp.Diagnostics.AddError("invalid-import-id", "scope must be 'troop' or 'site'")
		return
	}
	if game != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("game_id"), game)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("axis"), axis)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// write PUTs the preset (idempotent upsert) and refreshes the computed +
// server-canonical fields back into the model.
func (r *presetResource) write(ctx context.Context, m *presetModel, diags *diag.Diagnostics) {
	valuesMap, err := jsonToMap(m.Values)
	if err != nil {
		diags.AddError("invalid-values-json", err.Error())
		return
	}
	var env presetEnvelope
	body := map[string]any{"values": valuesMap}
	if err := r.client.Put(ctx, presetPath(m.TroopID, m.SiteID, m.GameID, m.Axis.ValueString(), m.Name.ValueString()), body, &env); err != nil {
		diags.AddError("white-label-preset-write-failed", err.Error())
		return
	}
	if err := applyPresetWire(m, env.Preset); err != nil {
		diags.AddError("white-label-preset-decode-failed", err.Error())
	}
}

// applyPresetWire writes the server's canonical values + timestamp into the
// model. Scope / game / axis / name are identity (RequiresReplace) and are
// left as the plan/state set them.
func applyPresetWire(m *presetModel, w presetWire) error {
	vals, err := mapToNormalized(w.Values)
	if err != nil {
		return err
	}
	m.Values = vals
	m.UpdatedAt = types.StringValue(w.UpdatedAt)
	m.ID = types.StringValue(buildPresetID(*m))
	return nil
}
