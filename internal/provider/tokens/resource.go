// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package tokens

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
		Description: "A Caputchin management token (Personal Access Token). `type='account'` mints a master PAT (free, capped at 1 active per account per ADR-0028); `type='troop'` mints a troop-scoped PAT. Minting itself is free under the per-troop-axis seat model — token rows do not consume a seat. The seat is claimed at attach time (`caputchin_troop_pat`); each attached troop's non-revoked attachment count is capped at `accounts.seats_total - user_used`. The secret is returned only at creation; the resource stores it sensitively in state. Both `name` and `type` are immutable post-mint; changing either replaces the token (the management API does not support PATCH on tokens). Attach troop-PATs to specific troops via the separate `caputchin_troop_pat` resource.",
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
				Description: "Display-friendly prefix of the token value (first 16 chars).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret": schema.StringAttribute{
				Description: "Full bearer-token value. Returned ONCE at creation; never re-readable. Pipe to a secrets store immediately. Lost values cannot be recovered; destroy and recreate the resource to mint a new token.",
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

	resp.Diagnostics.Append(resp.State.Set(ctx, tokenModel{
		ID:        types.StringValue(env.Token.ID),
		Name:      types.StringValue(env.Token.Name),
		Type:      types.StringValue(env.Token.Type),
		Prefix:    types.StringValue(env.Token.Prefix),
		Secret:    types.StringValue(env.Token.Value),
		CreatedAt: types.Int64Value(env.Token.CreatedAt),
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
		// Preserve the secret; Read never returns it.
		resp.Diagnostics.Append(resp.State.Set(ctx, tokenModel{
			ID:        types.StringValue(row.ID),
			Name:      types.StringValue(row.Name),
			Type:      types.StringValue(row.Type),
			Prefix:    types.StringValue(row.Prefix),
			Secret:    state.Secret,
			CreatedAt: types.Int64Value(row.CreatedAt),
		})...)
		return
	}
	resp.State.RemoveResource(ctx)
}

// Update is intentionally a no-op surface: every mutable attribute on the
// schema is marked RequiresReplace, so the framework routes any change
// through Create/Delete rather than Update. The method exists because the
// resource.Resource interface requires it.
func (r *tokenResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
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
	// prefix, created_at) but the secret is unrecoverable; destroy and
	// recreate the resource to mint a usable token.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret"), types.StringNull())...)
}
