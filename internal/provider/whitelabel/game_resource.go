// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

var (
	_ resource.Resource                     = (*gameResource)(nil)
	_ resource.ResourceWithConfigure        = (*gameResource)(nil)
	_ resource.ResourceWithConfigValidators = (*gameResource)(nil)
	_ resource.ResourceWithImportState      = (*gameResource)(nil)
)

// NewGameResource is the factory consumed by the provider's Resources() list.
func NewGameResource() resource.Resource {
	return &gameResource{}
}

type gameResource struct {
	client *client.Client
}

func (r *gameResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customized_game"
}

func (r *gameResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers a game in a scope's customized-games list. This is the REQUIRED parent for any game-axis `caputchin_white_label_preset` or `caputchin_custom_game_schema`: those reject with `game-not-registered` unless the game is registered first, so declare this resource and have the children set `game_id = caputchin_customized_game.<name>.game_id` (which also orders creation correctly). Set exactly one of `troop_id` / `site_id`. Requires the configuration tier (Solo+).\n\nWARNING: destroying this resource cascade-deletes the ENTIRE game customization for the scope: every preset (all axes) and every custom schema for the game, not just the registry row. When children reference its `game_id`, Terraform destroys them before this resource, so the cascade is a backstop; managing children for the same game WITHOUT that reference risks the cascade removing rows Terraform still tracks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Synthetic resource id encoding the composite key as `scope|id|game` (scope is troop|site). Matches the import id.",
				Computed:      true,
				PlanModifiers: useStateForUnknown(),
			},
			"troop_id": schema.StringAttribute{
				Description:   "Troop id. Exactly one of troop_id / site_id is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"site_id": schema.StringAttribute{
				Description:   "Site id. Exactly one of troop_id / site_id is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"game_id": schema.StringAttribute{
				Description:   "Game id (e.g. `caputchin/games/leaf` or a custom id). Forces replacement.",
				Required:      true,
				PlanModifiers: requiresReplace(),
			},
			"source": schema.StringAttribute{
				Description: "`marketplace` or `custom`. Omit to auto-derive from the marketplace catalog (present ⇒ marketplace, else custom).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("marketplace", "custom"),
				},
			},
			"auto_update": schema.BoolAttribute{
				Description: "When true, the install re-pins automatically when the indexer ships a newer version that passes the server-side replay check (non-destructive, server-driven). Default false. This is the declarative way to track the latest version. There is no one-shot \"update now\" attribute (Terraform is declarative): a manual re-pin is an imperative API action (`POST /game-customization/game/re-pin`) outside Terraform's model. Tainting + recreating also advances the pin, but it is destructive (the destroy cascade-deletes the game's presets + schemas first), so prefer auto_update.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"pinned_version": schema.StringAttribute{
				Description: "Vendored snapshot id this install serves. Null for custom / not-yet-vendored games. Read-only.",
				Computed:    true,
			},
			"update_available": schema.BoolAttribute{
				Description: "True when a newer self-check-passed version exists and auto_update is off. Read-only.",
				Computed:    true,
			},
		},
	}
}

// applyWire maps the management API game envelope onto the Terraform model.
func applyWire(m *gameModel, w gameWire) {
	m.Source = types.StringValue(w.Source)
	m.AutoUpdate = types.BoolValue(w.AutoUpdate)
	m.UpdateAvailable = types.BoolValue(w.UpdateAvailable)
	if w.PinnedVersionID != nil {
		m.PinnedVersion = types.StringValue(*w.PinnedVersionID)
	} else {
		m.PinnedVersion = types.StringNull()
	}
}

func (r *gameResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("troop_id"), path.MatchRoot("site_id")),
	}
}

func (r *gameResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *gameResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gameModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.register(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *gameResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gameModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var env gameEnvelope
	err := r.client.Get(ctx, gamePath(state.TroopID, state.SiteID, state.GameID), &env)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("customized-game-read-failed", err.Error())
		return
	}
	applyWire(&state, env.Game)
	state.ID = types.StringValue(buildGameID(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *gameResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gameModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only auto_update is mutable in place (game_id / scope force replacement).
	// PATCH it rather than re-register: a re-register would re-pin to the current
	// version, which is the "manual update" semantics, not a preference toggle.
	body := map[string]any{"auto_update": plan.AutoUpdate.ValueBool()}
	var env gameEnvelope
	if err := r.client.Patch(ctx, gamePath(plan.TroopID, plan.SiteID, plan.GameID), body, &env); err != nil {
		resp.Diagnostics.AddError("customized-game-update-failed", err.Error())
		return
	}
	applyWire(&plan, env.Game)
	plan.ID = types.StringValue(buildGameID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *gameResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gameModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Cascade delete. Removes every preset + schema + the registry row.
	err := r.client.Delete(ctx, gamePath(state.TroopID, state.SiteID, state.GameID))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("customized-game-delete-failed", err.Error())
	}
}

// Import id format: scope|id|game.
func (r *gameResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *gameResource) register(ctx context.Context, m *gameModel, diags *diag.Diagnostics) {
	body := map[string]any{"game": m.GameID.ValueString()}
	if isSet(m.Source) {
		body["source"] = m.Source.ValueString()
	}
	if !m.AutoUpdate.IsNull() && !m.AutoUpdate.IsUnknown() {
		body["auto_update"] = m.AutoUpdate.ValueBool()
	}
	var env gameEnvelope
	if err := r.client.Post(ctx, gamesPath(m.TroopID, m.SiteID), body, &env); err != nil {
		// Register is create-once: the API 409s a game already registered for the
		// scope (e.g. registered out-of-band via the dashboard). Point the user at
		// import rather than letting the generic error obscure the cause.
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "game-already-registered" {
			diags.AddError(
				"customized-game-already-registered",
				fmt.Sprintf("Game %q is already registered for this scope. Import it instead: terraform import <resource> '%s'.", m.GameID.ValueString(), buildGameID(*m)),
			)
			return
		}
		diags.AddError("customized-game-register-failed", err.Error())
		return
	}
	applyWire(m, env.Game)
	m.ID = types.StringValue(buildGameID(*m))
}
