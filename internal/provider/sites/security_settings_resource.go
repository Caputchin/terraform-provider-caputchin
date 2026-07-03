// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"fmt"

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
	return &siteSecuritySettingsResource{}
}

type siteSecuritySettingsResource struct {
	client *client.Client
}

func (r *siteSecuritySettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_security_settings"
}

func (r *siteSecuritySettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Per-site security settings for a Caputchin site key (the game gate). Singleton: one row per site.\n\nWhen `require_game` is true, verification on this site key must be gated by a game the server replays, instead of proof-of-work only. Enabling requires at least one installed marketplace game with a replayable artifact for this site (its own or inherited from the troop); otherwise the API rejects the change.\n\n`preview_mode` is a development/integration aid, nullable to support inheritance: when the effective value (this site's own setting, or the troop default when this is null) is true, the backend auto-approves every verification for this site key (no game, proof-of-work not enforced, `/siteverify` returns success), disabling bot protection while on. Sessions are still recorded, flagged preview.\n\nWhen `reuse` is true, one successful verification grants a short-lived clearance that lets later widget mounts skip replaying the game while the clearance is valid. `reuse_window_ms` bounds the clearance lifetime (server clamps to its own min/max regardless of the value set here); `reuse_persist` controls whether the clearance survives a page reload via a first-party cookie, versus staying in memory only. A troop-level `forbid_reuse` ceiling can force this off regardless of the site's own setting.\n\nWhen `proxy_gate` is true, the Proxy page-gate runs a full-page interstitial in front of the whole site at your reverse proxy (for hosts that cannot embed the widget); one solve mints a short-TTL gate pass cookie that clears later requests. `proxy_ttl_seconds` bounds the pass lifetime (server-clamped); `proxy_fail_mode` (`open`/`closed`) is advisory and templates the integration snippet. The Proxy page-gate requires the Alpha tier or higher; enabling it below that is rejected by the API. The advisory `proxy_path_scope` field is managed via the dashboard / API / MCP, not this resource.\n\nDestroying this resource removes Terraform tracking but does NOT reset the server-side setting.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "Identifier of the site these settings belong to. Changing this attribute forces replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"require_game": schema.BoolAttribute{
				Description: "If true, verification on this site key must be gated by a game (server-replayed); if false, verification is proof-of-work only and can be passed without playing a game.",
				Optional:    true,
				Computed:    true,
			},
			"preview_mode": schema.BoolAttribute{
				Description: "Preview mode for this site key: when effectively on the backend auto-approves every verification (no game, proof-of-work not enforced) and /siteverify returns success. A development/integration aid that DISABLES bot protection while on; sessions still record (flagged preview). null = inherit the troop default; effective value resolves site ?? troop ?? false.",
				Optional:    true,
				Computed:    true,
			},
			"reuse": schema.BoolAttribute{
				Description: "If true, one successful verification on this site key grants a short-lived clearance; later widget mounts present the clearance and skip replaying the game until it expires. A troop-level `forbid_reuse` ceiling overrides this to false.",
				Optional:    true,
				Computed:    true,
			},
			"reuse_window_ms": schema.Int64Attribute{
				Description: "Clearance lifetime in milliseconds while `reuse` is true. The server clamps this to its own min/max regardless of the value set here. May be null to use the server's default window.",
				Optional:    true,
				Computed:    true,
			},
			"reuse_persist": schema.BoolAttribute{
				Description: "If true (and `reuse` is true), the clearance survives a page reload via a first-party cookie; if false, it lives in memory only and is lost on reload.",
				Optional:    true,
				Computed:    true,
			},
			"proxy_gate": schema.BoolAttribute{
				Description: "If true, the Proxy page-gate is enabled: a full-page interstitial runs in front of the whole site at your reverse proxy, and one solve mints a short-TTL gate pass (a first-party cookie) that clears later requests. Requires the Alpha tier or higher; enabling on a lower tier is rejected by the API. The `proxy_path_scope` advisory setting is managed via the dashboard / API / MCP, not this resource.",
				Optional:    true,
				Computed:    true,
			},
			"proxy_ttl_seconds": schema.Int64Attribute{
				Description: "How long (seconds) one solve clears the visitor at the Proxy page-gate. The server clamps this to its own min/max regardless of the value set here. May be null to use the server's default.",
				Optional:    true,
				Computed:    true,
			},
			"proxy_fail_mode": schema.StringAttribute{
				Description: "What the reverse proxy should do when it can't reach the authorizer: `closed` blocks requests (safer for a login portal), `open` lets them through. Advisory — it templates the integration snippet; the proxy enforces it. May be null for the default (`closed`).",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *siteSecuritySettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *siteSecuritySettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteSecuritySettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildPatchBody(plan, siteSecuritySettingsModel{})
	if len(body) > 0 {
		if err := r.client.Patch(ctx, siteSecurityPath(plan.SiteID.ValueString()), body, nil); err != nil {
			resp.Diagnostics.AddError("site-security-create-failed", err.Error())
			return
		}
	}

	r.refreshState(ctx, plan.SiteID, &resp.Diagnostics, func(m siteSecuritySettingsModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

func (r *siteSecuritySettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteSecuritySettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.refreshState(ctx, state.SiteID, &resp.Diagnostics, func(m siteSecuritySettingsModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

func (r *siteSecuritySettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state siteSecuritySettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildPatchBody(plan, state)
	if len(body) > 0 {
		if err := r.client.Patch(ctx, siteSecurityPath(plan.SiteID.ValueString()), body, nil); err != nil {
			resp.Diagnostics.AddError("site-security-update-failed", err.Error())
			return
		}
	}

	r.refreshState(ctx, plan.SiteID, &resp.Diagnostics, func(m siteSecuritySettingsModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

// Delete removes the resource from Terraform state only. There is no
// server-side delete for a singleton setting (the site keeps its current value).
func (r *siteSecuritySettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *siteSecuritySettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import id IS the site id (the resource is singleton-per-site).
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), req.ID)...)
}

func siteSecurityPath(siteID string) string {
	return "/v1/management/sites/" + siteID + "/security-settings"
}

func (r *siteSecuritySettingsResource) refreshState(ctx context.Context, siteIDTF types.String, diags diagSink, sink func(siteSecuritySettingsModel)) {
	siteID := siteIDTF.ValueString()
	var env siteSecuritySettingsEnvelope
	if err := r.client.Get(ctx, siteSecurityPath(siteID), &env); err != nil {
		diags.AddError("site-security-read-failed", err.Error())
		return
	}
	sink(env.Settings.toModel(siteID))
}

// buildPatchBody emits only the fields that differ from prior state.
// On Create, state is the zero-value model; a set field is a change.
func (r *siteSecuritySettingsResource) buildPatchBody(plan, state siteSecuritySettingsModel) map[string]any {
	body := map[string]any{}
	if changedBool(plan.RequireGame, state.RequireGame) {
		body["require_game"] = plan.RequireGame.ValueBool()
	}
	if changedBoolNullable(plan.PreviewMode, state.PreviewMode) {
		if plan.PreviewMode.IsNull() {
			body["preview_mode"] = nil
		} else {
			body["preview_mode"] = plan.PreviewMode.ValueBool()
		}
	}
	if changedBool(plan.Reuse, state.Reuse) {
		body["reuse"] = plan.Reuse.ValueBool()
	}
	if changedIntNullable(plan.ReuseWindowMs, state.ReuseWindowMs) {
		if plan.ReuseWindowMs.IsNull() {
			body["reuse_window_ms"] = nil
		} else {
			body["reuse_window_ms"] = plan.ReuseWindowMs.ValueInt64()
		}
	}
	if changedBool(plan.ReusePersist, state.ReusePersist) {
		body["reuse_persist"] = plan.ReusePersist.ValueBool()
	}
	if changedBool(plan.ProxyGate, state.ProxyGate) {
		body["proxy_gate"] = plan.ProxyGate.ValueBool()
	}
	if changedIntNullable(plan.ProxyTtlSeconds, state.ProxyTtlSeconds) {
		if plan.ProxyTtlSeconds.IsNull() {
			body["proxy_ttl_seconds"] = nil
		} else {
			body["proxy_ttl_seconds"] = plan.ProxyTtlSeconds.ValueInt64()
		}
	}
	if changedStringNullable(plan.ProxyFailMode, state.ProxyFailMode) {
		if plan.ProxyFailMode.IsNull() {
			body["proxy_fail_mode"] = nil
		} else {
			body["proxy_fail_mode"] = plan.ProxyFailMode.ValueString()
		}
	}
	return body
}
