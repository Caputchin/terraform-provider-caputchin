// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewMemberResource is the factory for caputchin_troop_member.
func NewMemberResource() resource.Resource {
	return &memberResource{}
}

type memberResource struct {
	client *client.Client
}

type memberModel struct {
	ID               types.String   `tfsdk:"id"`
	TroopID          types.String   `tfsdk:"troop_id"`
	Email            types.String   `tfsdk:"email"`
	AccountID        types.String   `tfsdk:"account_id"`
	Perms            *patPermsModel `tfsdk:"perms"`
	Scope            *patScopeModel `tfsdk:"scope"`
	WouldConsumeSeat types.Bool     `tfsdk:"would_consume_seat"`
}

// apiUserMembership is the wire shape returned by GET /troops/{id}/members
// and POST /troops/{id}/members (inside a `member` envelope on POST). The
// email field is typed as a pointer because the platform route returns
// it as `null` when the underlying account row has been deleted between
// the membership row and the response shaping. A null email reaching
// the membership Read path is treated as a vanished member and the
// resource is removed from state (rather than letting `""` overwrite
// the user-supplied required-replace attribute, which would force a
// recreate on the next plan with seat churn).
type apiUserMembership struct {
	ID        string  `json:"id"`
	TroopID   string  `json:"troop_id"`
	AccountID string  `json:"account_id"`
	Email     *string `json:"email"`
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

type memberEnvelope struct {
	Member           apiUserMembership `json:"member"`
	WouldConsumeSeat bool              `json:"would_consume_seat"`
}

type memberListEnvelope struct {
	Members []apiUserMembership `json:"members"`
}

func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_troop_member"
}

func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A user membership in a Caputchin troop. Invitees are identified by email; the membership row is created idempotently for an existing or newly-invited account. Consumes one user seat if the invitee is not already a member of any troop in this account (sharing across troops is free). `troop_id` and `email` are immutable; changing either forces replacement (the underlying row is per-(troop, account) pair). `perms` and `scope` are mutable in place. Stripping every perm is refused server-side (`no-perms-granted`); destroy the resource to remove the member.",
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
			"email": schema.StringAttribute{
				Description: "Invitee email. Immutable (the membership row is keyed by the troop+account pair).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_id": schema.StringAttribute{
				Description: "Account id of the invitee (resolved server-side by email).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"would_consume_seat": schema.BoolAttribute{
				Description: "Whether this membership consumed a fresh user seat at creation time (sharing across troops within the same account is free, so subsequent memberships for an account already in another troop here come back as `false`). Snapshot at Create; preserved through subsequent reads (the management API does not echo it on GET).",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"perms": schema.SingleNestedAttribute{
				Description: "Permission set. Stripping every perm is refused.",
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
						Validators: []validator.String{
							stringvalidator.OneOf("full", "partial"),
						},
					},
					"site_ids": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Description: "Per-site allowlist. Required (non-empty) when `kind=\"partial\"`; ignored otherwise.",
					},
				},
			},
		},
	}
}

func (r *memberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan memberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildMemberRequestBody(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	body["email"] = plan.Email.ValueString()

	var env memberEnvelope
	if err := r.client.Post(ctx, "/v1/management/troops/"+plan.TroopID.ValueString()+"/members", body, &env); err != nil {
		resp.Diagnostics.AddError("troop-member-add-failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, memberMembershipToModel(ctx, env.Member, types.BoolValue(env.WouldConsumeSeat), &resp.Diagnostics))...)
}

func (r *memberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state memberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var env memberListEnvelope
	if err := r.client.Get(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/members", &env); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("troop-member-read-failed", err.Error())
		return
	}

	id := state.ID.ValueString()
	for _, m := range env.Members {
		if m.ID == id {
			if m.Email == nil {
				// Account row vanished; treat as deleted to avoid
				// silently overwriting the user-supplied
				// RequiresReplace email with an empty string.
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, memberMembershipToModel(ctx, m, state.WouldConsumeSeat, &resp.Diagnostics))...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *memberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state memberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildMemberRequestBody(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Patch(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/members/"+state.ID.ValueString(), body, nil); err != nil {
		resp.Diagnostics.AddError("troop-member-update-failed", err.Error())
		return
	}

	// PATCH returns {updated: true}; re-list to refresh computed fields.
	var env memberListEnvelope
	if err := r.client.Get(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/members", &env); err != nil {
		resp.Diagnostics.AddError("troop-member-refresh-failed", err.Error())
		return
	}
	id := state.ID.ValueString()
	for _, m := range env.Members {
		if m.ID == id {
			if m.Email == nil {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, memberMembershipToModel(ctx, m, state.WouldConsumeSeat, &resp.Diagnostics))...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *memberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state memberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/v1/management/troops/"+state.TroopID.ValueString()+"/members/"+state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("troop-member-remove-failed", err.Error())
	}
}

func (r *memberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
	// would_consume_seat is a Create-time signal; the API doesn't echo
	// it on GET, so we cannot reconstruct it from an imported row.
	// Leave as null; future plans will see Unknown and the
	// UseStateForUnknown plan modifier carries it forward without a
	// spurious diff.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("would_consume_seat"), types.BoolNull())...)
}
