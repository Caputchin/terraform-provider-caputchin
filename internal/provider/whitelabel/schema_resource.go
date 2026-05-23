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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

var (
	_ resource.Resource                     = (*schemaResource)(nil)
	_ resource.ResourceWithConfigure        = (*schemaResource)(nil)
	_ resource.ResourceWithConfigValidators = (*schemaResource)(nil)
	_ resource.ResourceWithImportState      = (*schemaResource)(nil)
)

// NewSchemaResource is the factory consumed by the provider's Resources() list.
func NewSchemaResource() resource.Resource {
	return &schemaResource{}
}

type schemaResource struct {
	client *client.Client
}

func (r *schemaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_game_schema"
}

func (r *schemaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Per-axis editable-field schema for a custom (non-marketplace) game, declaring the keys customers may override (ADR-0061). Set exactly one of `troop_id` / `site_id`. Carries no plan-tier gate (it is metadata describing the shape of presets). Changing scope, game, or axis forces replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Synthetic resource id encoding the composite key as `scope|id|game|axis` (scope is troop|site). Matches the import id.",
				Computed:      true,
				PlanModifiers: useStateForUnknown(),
			},
			"troop_id": schema.StringAttribute{
				Description:   "Troop id for a troop-wide schema. Exactly one of troop_id / site_id is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"site_id": schema.StringAttribute{
				Description:   "Site id for a per-site schema. Exactly one of troop_id / site_id is required. Forces replacement.",
				Optional:      true,
				PlanModifiers: requiresReplace(),
			},
			"game_id": schema.StringAttribute{
				Description:   "Game id (e.g. a custom id not in the marketplace) the schema describes. Forces replacement.",
				Required:      true,
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
			"schema": schema.StringAttribute{
				Description: "JSON object declaring the axis's editable fields (use `jsonencode({...})`). Compared semantically.",
				Required:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
			"updated_at": schema.StringAttribute{
				Description: "Server timestamp (ISO 8601) of the last write.",
				Computed:    true,
			},
		},
	}
}

func (r *schemaResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("troop_id"), path.MatchRoot("site_id")),
	}
}

func (r *schemaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *schemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan schemaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.write(ctx, &plan, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *schemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state schemaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var env schemaEnvelope
	err := r.client.Get(ctx, schemaPath(state.TroopID, state.SiteID, state.GameID, state.Axis.ValueString()), &env)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("custom-game-schema-read-failed", err.Error())
		return
	}
	if err := applySchemaWire(&state, env.Schema); err != nil {
		resp.Diagnostics.AddError("custom-game-schema-decode-failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *schemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan schemaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.write(ctx, &plan, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *schemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state schemaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.Delete(ctx, schemaPath(state.TroopID, state.SiteID, state.GameID, state.Axis.ValueString()))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("custom-game-schema-delete-failed", err.Error())
	}
}

// Import id format: scope|id|game|axis (game required).
func (r *schemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "|", 4)
	if len(parts) != 4 {
		resp.Diagnostics.AddError("invalid-import-id", "expected scope|id|game|axis (scope is troop|site)")
		return
	}
	scope, id, game, axis := parts[0], parts[1], parts[2], parts[3]
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("axis"), axis)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *schemaResource) write(ctx context.Context, m *schemaModel, diags *diag.Diagnostics) {
	schemaMap, err := jsonToMap(m.Schema)
	if err != nil {
		diags.AddError("invalid-schema-json", err.Error())
		return
	}
	var env schemaEnvelope
	body := map[string]any{"schema": schemaMap}
	if err := r.client.Put(ctx, schemaPath(m.TroopID, m.SiteID, m.GameID, m.Axis.ValueString()), body, &env); err != nil {
		diags.AddError("custom-game-schema-write-failed", err.Error())
		return
	}
	if err := applySchemaWire(m, env.Schema); err != nil {
		diags.AddError("custom-game-schema-decode-failed", err.Error())
	}
}

func applySchemaWire(m *schemaModel, w schemaWire) error {
	s, err := mapToNormalized(w.Schema)
	if err != nil {
		return err
	}
	m.Schema = s
	m.UpdatedAt = types.StringValue(w.UpdatedAt)
	m.ID = types.StringValue(buildSchemaID(*m))
	return nil
}
