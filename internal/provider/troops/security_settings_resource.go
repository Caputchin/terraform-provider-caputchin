// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewSecuritySettingsResource is the factory consumed by the provider's Resources() list.
func NewSecuritySettingsResource() resource.Resource {
	return &troopSecuritySettingsResource{}
}

type troopSecuritySettingsResource struct {
	client *client.Client
}

func (r *troopSecuritySettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_troop_security_settings"
}

func (r *troopSecuritySettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Troop-wide security settings (the game-gate ceiling). Singleton: one row per troop.\n\nWhen `force_game` is true, every site key in the troop is gated by a game regardless of each site's own setting. Enabling requires at least one installed troop-level marketplace game with a replayable artifact; otherwise the API rejects the change. Requires full-scope create|edit on the troop.\n\n`preview_mode` sets the troop-wide default inherited by any site key that leaves its own `preview_mode` null (site ?? troop ?? false). When the effective value is true, a gated site key still serves its normal experience (the game and its shells/chrome), but the backend auto-approves every verification regardless of the solve (game replay and cap not enforced), disabling bot protection while on. Sessions are still recorded, flagged preview. `null` here is equivalent to false.\n\nVerification reuse (`reuse`, `reuse_window_ms`, `reuse_persist`) sets the troop-wide default inherited by any site key that leaves its own value null (site ?? troop ?? default). When effectively on, one solve grants a short-lived clearance that lets later widget mounts skip replaying the game while it is valid; `reuse_window_ms` bounds the clearance lifetime (server-clamped) and `reuse_persist` stores it in a first-party cookie versus memory only. `null` here means no troop default.\n\nDestroying this resource removes Terraform tracking but does NOT reset the server-side setting.",
		Attributes: map[string]schema.Attribute{
			"troop_id": schema.StringAttribute{
				Description: "Identifier of the troop these settings belong to. Changing this attribute forces replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"force_game": schema.BoolAttribute{
				Description: "If true, every site key in the troop must gate verification with a game, regardless of its own setting.",
				Optional:    true,
				Computed:    true,
			},
			"preview_mode": schema.BoolAttribute{
				Description: "Troop-wide preview-mode default inherited by any site key that sets none (site ?? troop ?? false). When effectively on, those site keys still serve their normal experience (the game and its shells/chrome when gated, plain cap otherwise), but the backend auto-approves every verification regardless of the solve (game replay and cap not enforced), disabling bot protection while on. null is equivalent to false.",
				Optional:    true,
				Computed:    true,
			},
			"reuse": schema.BoolAttribute{
				Description: "Troop-wide verification-reuse default inherited by any site key that sets none (site ?? troop ?? false). When effectively on, one solve grants a short-lived clearance so later widget mounts skip replaying the game. null means no troop default.",
				Optional:    true,
				Computed:    true,
			},
			"reuse_window_ms": schema.Int64Attribute{
				Description: "Troop-wide default clearance lifetime in milliseconds, inherited by any site key that sets none. The server clamps this to its own min/max. null means no troop default (the server default applies).",
				Optional:    true,
				Computed:    true,
			},
			"reuse_persist": schema.BoolAttribute{
				Description: "Troop-wide default for whether the reuse clearance survives a page reload via a first-party cookie, inherited by any site key that sets none. null is equivalent to false.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *troopSecuritySettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected-provider-data",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *troopSecuritySettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan troopSecuritySettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildPatchBody(plan, troopSecuritySettingsModel{})
	if len(body) > 0 {
		if err := r.client.Patch(ctx, troopSecurityPath(plan.TroopID.ValueString()), body, nil); err != nil {
			resp.Diagnostics.AddError("troop-security-create-failed", err.Error())
			return
		}
	}

	r.refreshState(ctx, plan.TroopID, &resp.Diagnostics, func(m troopSecuritySettingsModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

func (r *troopSecuritySettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state troopSecuritySettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.refreshState(ctx, state.TroopID, &resp.Diagnostics, func(m troopSecuritySettingsModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

func (r *troopSecuritySettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state troopSecuritySettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildPatchBody(plan, state)
	if len(body) > 0 {
		if err := r.client.Patch(ctx, troopSecurityPath(plan.TroopID.ValueString()), body, nil); err != nil {
			resp.Diagnostics.AddError("troop-security-update-failed", err.Error())
			return
		}
	}

	r.refreshState(ctx, plan.TroopID, &resp.Diagnostics, func(m troopSecuritySettingsModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

// Delete removes the resource from Terraform state only. There is no
// server-side delete for a singleton setting (the troop keeps its current value).
func (r *troopSecuritySettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *troopSecuritySettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import id IS the troop id (the resource is singleton-per-troop).
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("troop_id"), req.ID)...)
}

func troopSecurityPath(troopID string) string {
	return "/v1/management/troops/" + troopID + "/security-settings"
}

func (r *troopSecuritySettingsResource) refreshState(ctx context.Context, troopIDTF types.String, diags *diag.Diagnostics, sink func(troopSecuritySettingsModel)) {
	troopID := troopIDTF.ValueString()
	var env troopSecuritySettingsEnvelope
	if err := r.client.Get(ctx, troopSecurityPath(troopID), &env); err != nil {
		diags.AddError("troop-security-read-failed", err.Error())
		return
	}
	sink(env.Settings.toModel(troopID))
}

// buildPatchBody emits only the fields that differ from prior state.
func (r *troopSecuritySettingsResource) buildPatchBody(plan, state troopSecuritySettingsModel) map[string]any {
	body := map[string]any{}
	if changedBool(plan.ForceGame, state.ForceGame) {
		body["force_game"] = plan.ForceGame.ValueBool()
	}
	if changedBool(plan.PreviewMode, state.PreviewMode) {
		body["preview_mode"] = plan.PreviewMode.ValueBool()
	}
	if changedBoolNullable(plan.Reuse, state.Reuse) {
		if plan.Reuse.IsNull() {
			body["reuse"] = nil
		} else {
			body["reuse"] = plan.Reuse.ValueBool()
		}
	}
	if changedIntNullable(plan.ReuseWindowMs, state.ReuseWindowMs) {
		if plan.ReuseWindowMs.IsNull() {
			body["reuse_window_ms"] = nil
		} else {
			body["reuse_window_ms"] = plan.ReuseWindowMs.ValueInt64()
		}
	}
	if changedBoolNullable(plan.ReusePersist, state.ReusePersist) {
		if plan.ReusePersist.IsNull() {
			body["reuse_persist"] = nil
		} else {
			body["reuse_persist"] = plan.ReusePersist.ValueBool()
		}
	}
	return body
}
