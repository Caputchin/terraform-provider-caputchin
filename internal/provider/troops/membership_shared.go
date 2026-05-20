// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/util"
)

// buildPatRequestBody renders the perms + scope sub-objects into the
// shape the management API expects on POST /troops/{id}/pats and PATCH
// /troops/{id}/pats/{membershipId}. Errors land on the supplied
// diagnostics rather than returning an `error`, matching the surrounding
// resource code style.
func buildPatRequestBody(ctx context.Context, plan patModel, diags *diag.Diagnostics) map[string]any {
	body := map[string]any{}
	if plan.Perms != nil {
		body["perms"] = map[string]bool{
			"create": plan.Perms.Create.ValueBool(),
			"edit":   plan.Perms.Edit.ValueBool(),
			"read":   plan.Perms.Read.ValueBool(),
			"manage": plan.Perms.Manage.ValueBool(),
		}
	}
	if plan.Scope != nil {
		scope := map[string]any{"kind": plan.Scope.Kind.ValueString()}
		if plan.Scope.Kind.ValueString() == "partial" {
			var siteIDs []string
			diags.Append(plan.Scope.SiteIDs.ElementsAs(ctx, &siteIDs, false)...)
			scope["site_ids"] = siteIDs
		}
		body["scope"] = scope
	}
	return body
}

// buildMemberRequestBody is the user-membership counterpart of
// buildPatRequestBody. Same perms + scope sub-objects.
func buildMemberRequestBody(ctx context.Context, plan memberModel, diags *diag.Diagnostics) map[string]any {
	body := map[string]any{}
	if plan.Perms != nil {
		body["perms"] = map[string]bool{
			"create": plan.Perms.Create.ValueBool(),
			"edit":   plan.Perms.Edit.ValueBool(),
			"read":   plan.Perms.Read.ValueBool(),
			"manage": plan.Perms.Manage.ValueBool(),
		}
	}
	if plan.Scope != nil {
		scope := map[string]any{"kind": plan.Scope.Kind.ValueString()}
		if plan.Scope.Kind.ValueString() == "partial" {
			var siteIDs []string
			diags.Append(plan.Scope.SiteIDs.ElementsAs(ctx, &siteIDs, false)...)
			scope["site_ids"] = siteIDs
		}
		body["scope"] = scope
	}
	return body
}

// patMembershipToModel projects an apiPatMembership wire object into the
// Terraform state shape.
func patMembershipToModel(ctx context.Context, m apiPatMembership, diags *diag.Diagnostics) patModel {
	siteIDsList, listDiags := types.ListValueFrom(ctx, types.StringType, m.Scope.SiteIDs)
	diags.Append(listDiags...)
	return patModel{
		ID:        types.StringValue(m.ID),
		TroopID:   types.StringValue(m.TroopID),
		PatID:     types.StringValue(m.PatID),
		PatName:   util.NullableString(m.PatName),
		PatPrefix: util.NullableString(m.PatPrefix),
		Perms: &patPermsModel{
			Create: types.BoolValue(m.Perms.Create),
			Edit:   types.BoolValue(m.Perms.Edit),
			Read:   types.BoolValue(m.Perms.Read),
			Manage: types.BoolValue(m.Perms.Manage),
		},
		Scope: &patScopeModel{
			Kind:    types.StringValue(m.Scope.Kind),
			SiteIDs: siteIDsList,
		},
	}
}

// memberMembershipToModel projects an apiUserMembership wire object into
// the Terraform state shape. Callers must check `m.Email == nil` and
// remove the resource from state before invoking this — a null email
// from the wire indicates a vanished account and the membership should
// not be projected into state.
//
// `wouldConsumeSeat` is a Create-time signal that the API echoes only
// on POST /troops/{id}/members; subsequent GETs do not include it. The
// resource Read/Update flow passes the prior state value through this
// helper so the attribute survives refreshes.
func memberMembershipToModel(ctx context.Context, m apiUserMembership, wouldConsumeSeat types.Bool, diags *diag.Diagnostics) memberModel {
	siteIDsList, listDiags := types.ListValueFrom(ctx, types.StringType, m.Scope.SiteIDs)
	diags.Append(listDiags...)
	return memberModel{
		ID:        types.StringValue(m.ID),
		TroopID:   types.StringValue(m.TroopID),
		Email:     util.NullableString(m.Email),
		AccountID: types.StringValue(m.AccountID),
		Perms: &patPermsModel{
			Create: types.BoolValue(m.Perms.Create),
			Edit:   types.BoolValue(m.Perms.Edit),
			Read:   types.BoolValue(m.Perms.Read),
			Manage: types.BoolValue(m.Perms.Manage),
		},
		Scope: &patScopeModel{
			Kind:    types.StringValue(m.Scope.Kind),
			SiteIDs: siteIDsList,
		},
		WouldConsumeSeat: wouldConsumeSeat,
	}
}

// splitImportID parses a colon-delimited compound import id used by
// resources keyed by (troop_id, membership_id).
func splitImportID(s string) []string {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}
	return parts
}
