// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

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

// NewPatResource is the factory for caputchin_troop_pat.
func NewPatResource() resource.Resource {
	return &patResource{}
}

type patResource struct {
	client *client.Client
}

// patPermsModel matches the API's perms object: create / edit / read /
// manage booleans. All four are Required so plans are unambiguous —
// stripping every perm is refused server-side with `no-perms-granted`.
type patPermsModel struct {
	Create types.Bool `tfsdk:"create"`
	Edit   types.Bool `tfsdk:"edit"`
	Read   types.Bool `tfsdk:"read"`
	Manage types.Bool `tfsdk:"manage"`
}

// patScopeModel matches the API's scope discriminated union via two
// fields: `kind` is "full" or "partial"; `site_ids` is the per-site
// allowlist when kind=partial and is ignored (kept null/empty) when
// kind=full. The provider enforces the contract on the wire — Terraform
// does not natively express discriminated unions.
type patScopeModel struct {
	Kind    types.String `tfsdk:"kind"`
	SiteIDs types.List   `tfsdk:"site_ids"`
}

type patModel struct {
	ID        types.String   `tfsdk:"id"`
	TroopID   types.String   `tfsdk:"troop_id"`
	PatID     types.String   `tfsdk:"pat_id"`
	PatName   types.String   `tfsdk:"pat_name"`
	PatPrefix types.String   `tfsdk:"pat_prefix"`
	Perms     *patPermsModel `tfsdk:"perms"`
	Scope     *patScopeModel `tfsdk:"scope"`
}

// apiPatMembership is the wire shape returned by GET /troops/{id}/pats
// and POST /troops/{id}/pats (inside a `pat` envelope on POST). The
// pat_name and pat_prefix fields are typed as pointers because the
// platform routes return them as `null` when the underlying token row
// has been revoked between the membership row and the response shaping
// (rare; the membership row outlives a soft-revoked token). A null
// decode lands as a nil pointer; the conversion helper projects nil
// into `types.StringNull()` rather than `""`.
type apiPatMembership struct {
	ID        string  `json:"id"`
	TroopID   string  `json:"troop_id"`
	PatID     string  `json:"pat_id"`
	PatName   *string `json:"pat_name"`
	PatPrefix *string `json:"pat_prefix"`
	Perms     struct {
		Create bool `json:"create"`
		Edit   bool `json:"edit"`
		Read   bool `json:"read"`
		Manage bool `json:"manage"`
	} `json:"perms"`
	Scope struct {
		Kind    string   `json:"kind"`
		SiteIDs []string `json:"site_ids"`
	} `json:"scope"`
}

type patEnvelope struct {
	Pat apiPatMembership `json:"pat"`
}

type patListEnvelope struct {
	Pats []apiPatMembership `json:"pats"`
}

func (r *patResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_troop_pat"
}

func (r *patResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches an existing troop-PAT (minted via `caputchin_account_token` with `type=\"troop\"`) to a specific troop with a permission set and scope. No seat is consumed here (the PAT seat was claimed at mint time), and the same PAT may be attached to multiple troops with different permissions. `troop_id` and `pat_id` are immutable; changing either forces replacement (the underlying membership row is per-(troop, pat) pair). `perms` and `scope` are mutable in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Server-issued membership identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"troop_id": schema.StringAttribute{
				Description: "Owning troop identifier. Immutable.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pat_id": schema.StringAttribute{
				Description: "Token identifier (from `caputchin_account_token` with `type=\"troop\"`). Immutable.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pat_name": schema.StringAttribute{
				Description: "Name of the attached token (snapshot of `caputchin_account_token.name`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pat_prefix": schema.StringAttribute{
				Description: "Display prefix of the attached token (snapshot of `caputchin_account_token.prefix`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"perms": schema.SingleNestedAttribute{
				Description: "Permission set. Stripping every perm is refused (`no-perms-granted`); destroy the resource to remove the membership instead.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"create": schema.BoolAttribute{Required: true, Description: "May create site keys / configs in scope."},
					"edit":   schema.BoolAttribute{Required: true, Description: "May edit existing site keys / configs in scope."},
					"read":   schema.BoolAttribute{Required: true, Description: "May read site keys / configs / stats in scope."},
					"manage": schema.BoolAttribute{Required: true, Description: "May manage the troop's members and PAT memberships."},
				},
			},
			"scope": schema.SingleNestedAttribute{
				Description: "Site scope. `kind=\"full\"` grants the perms across all sites in the troop; `kind=\"partial\"` restricts them to the listed `site_ids` (at least one).",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"kind": schema.StringAttribute{
						Required:    true,
						Description: "`full` or `partial`.",
					},
					"site_ids": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Description: "Per-site allowlist. Required (non-empty) when `kind=\"partial\"`; ignored otherwise. The server echoes an empty list when `kind=\"full\"`.",
					},
				},
			},
		},
	}
}

func (r *patResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *patResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan patModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildPatRequestBody(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	body["pat_id"] = plan.PatID.ValueString()

	var env patEnvelope
	if err := r.client.Post(ctx, "/v1/management/troops/"+plan.TroopID.ValueString()+"/pats", body, &env); err != nil {
		resp.Diagnostics.AddError("troop-pat-attach-failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, patMembershipToModel(ctx, env.Pat, &resp.Diagnostics))...)
}

func (r *patResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state patModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// API has no GET-by-id; list the troop's pat memberships and filter
	// by membership id. Missing row → resource removed from state.
	var env patListEnvelope
	if err := r.client.Get(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/pats", &env); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("troop-pat-read-failed", err.Error())
		return
	}

	id := state.ID.ValueString()
	for _, m := range env.Pats {
		if m.ID == id {
			resp.Diagnostics.Append(resp.State.Set(ctx, patMembershipToModel(ctx, m, &resp.Diagnostics))...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *patResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state patModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildPatRequestBody(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Patch(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/pats/"+state.ID.ValueString(), body, nil); err != nil {
		resp.Diagnostics.AddError("troop-pat-update-failed", err.Error())
		return
	}

	// PATCH returns {updated: true}; re-read to refresh computed fields.
	var env patListEnvelope
	if err := r.client.Get(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/pats", &env); err != nil {
		resp.Diagnostics.AddError("troop-pat-refresh-failed", err.Error())
		return
	}
	id := state.ID.ValueString()
	for _, m := range env.Pats {
		if m.ID == id {
			resp.Diagnostics.Append(resp.State.Set(ctx, patMembershipToModel(ctx, m, &resp.Diagnostics))...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *patResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state patModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/pats/"+state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("troop-pat-detach-failed", err.Error())
	}
}

func (r *patResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import expects "<troop_id>:<membership_id>" — the membership row is
	// keyed by both because troop_id is path-scoped on every CRUD call.
	parts := splitImportID(req.ID)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"invalid-import-id",
			"Expected import id in the form `<troop_id>:<membership_id>`.",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("troop_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
