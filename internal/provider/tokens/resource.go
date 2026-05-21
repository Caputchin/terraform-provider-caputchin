// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package tokens

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewResource is the factory consumed by the provider's Resources() list.
func NewResource() resource.Resource {
	return &tokenResource{}
}

type tokenResource struct {
	client *client.Client
}

func (r *tokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_token"
}

func (r *tokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Caputchin management token (Personal Access Token). `type='account'` mints a master PAT (free, capped at 1 active per account per ADR-0028); `type='troop'` mints a troop-scoped PAT. Minting itself is free under the per-troop-axis seat model — token rows do not consume a seat. The seat is claimed at attach time (`caputchin_troop_pat`); each attached troop's non-revoked attachment count is capped at `accounts.seats_total - user_used`. The secret is returned only at creation; the resource stores it sensitively in state. Both `name` and `type` are immutable post-mint; changing either replaces the token (the management API does not support PATCH on tokens). To rotate the credential in place without losing troop attachments, bump `secret_version`. The provider issues POST /tokens/{id}/rotate; the row's `id` and `name` stay stable, the `prefix` rotates together with the secret half, and the rotated value lands in `secret` per ADR-0056. Attach troop-PATs to specific troops via the separate `caputchin_troop_pat` resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Server-issued token identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable token name. Immutable (the management API has no PATCH on tokens); changing this attribute forces replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Token type. `account` (master, capped at 1 active per account, free) or `troop` (per-troop-scope; mint is free, seat is claimed at attach time). Defaults to `troop`. Immutable; changing forces replacement.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("troop"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"prefix": schema.StringAttribute{
				Description: "Display-friendly prefix of the token value (first 16 chars). Rotates together with the secret half on every in-place rotation (ADR-0056) — refer to tokens across rotation by `id` or `name`, not `prefix`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret": schema.StringAttribute{
				Description: "Full bearer-token value. Returned ONCE at creation and on every in-place rotation (`secret_version` bump). Pipe to a secrets store immediately. The value is stored sensitively in state; treat the state file as secret-bearing. Lost values cannot be recovered; rotate via `secret_version` to mint a fresh value into state without destroying the resource.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.Int64Attribute{
				Description: "Creation timestamp in milliseconds since the Unix epoch.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64UseStateForUnknown{},
				},
			},
			"secret_version": schema.Int64Attribute{
				Description: "Provider-tracked rotation counter (ADR-0056). Bump the value to trigger an in-place credential rotation: the provider issues POST /tokens/{id}/rotate and writes the new value into the `secret` attribute. The token row's `id`, `name`, and `type` are unchanged; the `prefix` rotates together with the secret. Any troop attachments survive. Defaults to `0`; set explicitly to start at a different baseline. Initial Create does NOT call rotate regardless of the planned version (the mint already returns a fresh credential). Refused by the API if the calling token is the rotation target.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
		},
	}
}

func (r *tokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tokType := plan.Type.ValueString()
	if tokType == "" {
		tokType = "troop"
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
		"type": tokType,
	}
	var env createEnvelope
	if err := r.client.Post(ctx, "/v1/management/tokens", body, &env); err != nil {
		resp.Diagnostics.AddError("token-create-failed", err.Error())
		return
	}
	if env.Token.Value == "" {
		resp.Diagnostics.AddError(
			"missing-secret-on-create",
			"The management API did not return a token value for the newly minted token. This is a provider/API contract violation.",
		)
		return
	}

	// Persist the planned secret_version (or its default 0). Initial
	// Create does NOT call rotate regardless of the planned version (the
	// mint already returns a fresh secret); the provider just records
	// the version so future bumps fire the rotation branch in Update —
	// same shape as caputchin_site_key per ADR-0051 / ADR-0056.
	secretVersion := plan.SecretVersion
	if secretVersion.IsNull() || secretVersion.IsUnknown() {
		secretVersion = types.Int64Value(0)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, tokenModel{
		ID:            types.StringValue(env.Token.ID),
		Name:          types.StringValue(env.Token.Name),
		Type:          types.StringValue(env.Token.Type),
		Prefix:        types.StringValue(env.Token.Prefix),
		Secret:        types.StringValue(env.Token.Value),
		CreatedAt:     types.Int64Value(env.Token.CreatedAt),
		SecretVersion: secretVersion,
	})...)
}

func (r *tokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The management API exposes only GET /tokens (a list); there is no
	// GET-by-id endpoint. Refresh by listing and matching on id. Token
	// revocations are observed as a missing row (the list excludes
	// revoked tokens) and the resource is removed from state.
	var env listEnvelope
	if err := r.client.Get(ctx, "/v1/management/tokens", &env); err != nil {
		resp.Diagnostics.AddError("token-read-failed", err.Error())
		return
	}

	id := state.ID.ValueString()
	for _, row := range env.Tokens {
		if row.ID != id {
			continue
		}
		// Preserve the secret AND the provider-tracked secret_version;
		// the API has no notion of secret_version (lives entirely in
		// state), and Read never returns the secret value.
		resp.Diagnostics.Append(resp.State.Set(ctx, tokenModel{
			ID:            types.StringValue(row.ID),
			Name:          types.StringValue(row.Name),
			Type:          types.StringValue(row.Type),
			Prefix:        types.StringValue(row.Prefix),
			Secret:        state.Secret,
			CreatedAt:     types.Int64Value(row.CreatedAt),
			SecretVersion: state.SecretVersion,
		})...)
		return
	}
	resp.State.RemoveResource(ctx)
}

// Update branches on the provider-tracked `secret_version` (ADR-0056).
// Every other mutable attribute on the schema is RequiresReplace, so the
// framework routes name / type changes through Create + Delete; Update
// fires only when `secret_version` differs from state. In that case the
// provider POSTs to /v1/management/tokens/{id}/rotate, replaces the
// `secret` AND `prefix` in state with the rotated values from the
// response, and carries every other attribute forward from state. The
// token row's id and name are unchanged across rotation; the prefix
// rotates together with the secret half.
func (r *tokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plannedVersion := plan.SecretVersion
	if plannedVersion.IsNull() || plannedVersion.IsUnknown() {
		plannedVersion = state.SecretVersion
	}

	if plannedVersion.ValueInt64() == state.SecretVersion.ValueInt64() {
		// No-op Update — surfaces drift on a Computed field nothing
		// actually changed. Carry state forward unchanged.
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}

	var env rotateEnvelope
	if err := r.client.Post(ctx, "/v1/management/tokens/"+state.ID.ValueString()+"/rotate", map[string]any{}, &env); err != nil {
		resp.Diagnostics.AddError("token-rotate-failed", err.Error())
		return
	}
	if env.Token == "" {
		resp.Diagnostics.AddError(
			"missing-secret-on-rotate",
			"The management API did not return a token value from /tokens/{id}/rotate. This is a provider/API contract violation.",
		)
		return
	}

	// API roll prior to ADR-0056-update may not include `prefix` in the
	// rotate response; slice the bearer string locally as a fallback.
	// The token format is always `cpt_pat_<32 base64url>` = 40 chars and
	// `env.Token == ""` is rejected above, so len(env.Token) >= 16 holds
	// unconditionally and no stale-state-carry path is needed.
	rotatedPrefix := types.StringValue(env.Prefix)
	if env.Prefix == "" {
		rotatedPrefix = types.StringValue(env.Token[:16])
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, tokenModel{
		ID:            state.ID,
		Name:          state.Name,
		Type:          state.Type,
		Prefix:        rotatedPrefix,
		Secret:        types.StringValue(env.Token),
		CreatedAt:     state.CreatedAt,
		SecretVersion: plannedVersion,
	})...)
}

func (r *tokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, "/v1/management/tokens/"+state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("token-delete-failed", err.Error())
	}
}

func (r *tokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	// The imported row has no secret in state; the API never returns it
	// after the initial mint. Importing recovers metadata (name, type,
	// prefix, created_at). Customers recover by bumping `secret_version`
	// from `0` (null state) to `1` on the next plan; Update fires the
	// rotation branch and writes a fresh value into state without
	// destroying the resource — same shape as caputchin_site_key.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret"), types.StringNull())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret_version"), types.Int64Value(0))...)
}
