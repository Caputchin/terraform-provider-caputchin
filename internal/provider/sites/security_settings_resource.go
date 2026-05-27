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
		Description: "Per-site security settings for a Caputchin site key (the game gate). Singleton: one row per site.\n\nWhen `require_game` is true, verification on this site key must be gated by a game the server replays, instead of proof-of-work only. Enabling requires at least one installed marketplace game with a replayable artifact for this site (its own or inherited from the troop); otherwise the API rejects the change.\n\nDestroying this resource removes Terraform tracking but does NOT reset the server-side setting.",
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

// buildPatchBody emits require_game only when the plan differs from prior state.
// On Create, state is the zero-value model; a set require_game is a change.
func (r *siteSecuritySettingsResource) buildPatchBody(plan, state siteSecuritySettingsModel) map[string]any {
	body := map[string]any{}
	if changedBool(plan.RequireGame, state.RequireGame) {
		body["require_game"] = plan.RequireGame.ValueBool()
	}
	return body
}
