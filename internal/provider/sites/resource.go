// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewResource is the factory consumed by the provider's Resources() list.
func NewResource() resource.Resource {
	return &siteResource{}
}

type siteResource struct {
	client *client.Client
}

func (r *siteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_key"
}

func (r *siteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Caputchin site key — the public/secret pair a customer embeds in their site and uses to verify users. `troop_id` is immutable; changing it forces replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Server-issued site identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description: "Public site key (the value embedded in the widget on the customer's site).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable site name. Modifiable in place.",
				Required:    true,
			},
			"troop_id": schema.StringAttribute{
				Description: "Identifier of the owning troop. Immutable — changing this attribute forces replacement of the site key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tier": schema.StringAttribute{
				Description: "Plan tier inherited from the owning troop. Read-only.",
				Computed:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Whether the site key is disabled. Disabled keys still exist but reject verification calls. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"created_at": schema.Int64Attribute{
				Description: "Creation timestamp in milliseconds since the Unix epoch.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64UseStateForUnknown{},
				},
			},
			"secret": schema.StringAttribute{
				Description: "Secret used to authenticate server-side verification calls. Returned only at creation time; stored sensitively in state. Use the rotate-secret resource action to issue a new value (available in a later release).",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *siteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *siteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":     plan.Name.ValueString(),
		"troop_id": plan.TroopID.ValueString(),
	}
	var env siteEnvelope
	if err := r.client.Post(ctx, "/v1/management/sites", body, &env); err != nil {
		resp.Diagnostics.AddError("site-create-failed", err.Error())
		return
	}

	if env.Secret == "" {
		resp.Diagnostics.AddError(
			"missing-secret-on-create",
			"The management API did not return a secret for the newly created site key. This is a provider/API contract violation.",
		)
		return
	}

	state := env.Site.toModel(types.StringValue(env.Secret))
	// disabled defaults to false on create; the API does not echo it for fresh
	// rows because there's no `disabledAt` yet. The toModel projection already
	// reads bool from the wire (`false` zero-value when absent), which matches
	// the schema default.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *siteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var env siteEnvelope
	if err := r.client.Get(ctx, "/v1/management/sites/"+state.ID.ValueString(), &env); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("site-read-failed", err.Error())
		return
	}

	// Preserve the secret already in state — Read never returns it.
	refreshed := env.Site.toModel(state.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *siteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state siteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if plan.Name.ValueString() != state.Name.ValueString() {
		body["name"] = plan.Name.ValueString()
	}
	if plan.Disabled.ValueBool() != state.Disabled.ValueBool() {
		body["disabled"] = plan.Disabled.ValueBool()
	}

	if len(body) == 0 {
		// Nothing to send; just persist plan over state (e.g. drift in a
		// Computed field that nothing actually changed).
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		return
	}

	var env siteEnvelope
	if err := r.client.Patch(ctx, "/v1/management/sites/"+state.ID.ValueString(), body, &env); err != nil {
		resp.Diagnostics.AddError("site-update-failed", err.Error())
		return
	}

	refreshed := env.Site.toModel(state.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *siteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/v1/management/sites/"+state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("site-delete-failed", err.Error())
	}
}

func (r *siteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	// The imported row has no secret in state — Read won't populate it because
	// the API never returns it after the initial create. Customers must rotate
	// the secret to recover a usable value (rotate-secret action lands in a
	// later release; for v0.1 importers can issue a manual rotate via the
	// dashboard or OpenAPI client).
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret"), types.StringNull())...)
}
