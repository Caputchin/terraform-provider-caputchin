// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// toModel maps the wire shape into Terraform state for the singleton config
// resource. The caller supplies the site id (carried separately in the envelope).
// A diagSink is passed so list-conversion diagnostics surface to the caller's
// response.
func (cfg apiSiteConfig) toModel(ctx context.Context, siteID string, diags diagSink) siteConfigModel {
	headers, hd := stringListFromSlice(ctx, cfg.RequiredHeaders)
	cors, cd := stringListFromSlice(ctx, cfg.CorsOrigins)
	for _, d := range hd {
		diags.AddError("list-decode-failed", d)
	}
	for _, d := range cd {
		diags.AddError("list-decode-failed", d)
	}

	model := siteConfigModel{
		SiteID:                 types.StringValue(siteID),
		PowDifficulty:          types.Int64Value(cfg.Difficulty),
		PowChallengeCount:      types.Int64Value(cfg.ChallengeCount),
		ObfuscationLevel:       types.Int64Value(cfg.ObfuscationLevel),
		BlockAutomatedBrowsers: types.BoolValue(cfg.BlockAutomatedBrowsers),
		BlockNonBrowserUA:      nullableBool(cfg.BlockNonBrowserUA),
		RequiredHeaders:        headers,
		RatelimitMax:           nullableInt64(cfg.RatelimitMax),
		CorsOrigins:            cors,
	}
	return model
}

func nullableBool(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

func nullableInt64(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func stringListFromSlice(ctx context.Context, v []string) (types.List, []string) {
	if v == nil {
		return types.ListNull(types.StringType), nil
	}
	lv, d := types.ListValueFrom(ctx, types.StringType, v)
	return lv, diagToStrings(d)
}

func diagToStrings(d diag.Diagnostics) []string {
	if !d.HasError() {
		return nil
	}
	out := make([]string, 0, len(d))
	for _, e := range d.Errors() {
		out = append(out, e.Summary()+": "+e.Detail())
	}
	return out
}

func changedInt(plan, state types.Int64) bool {
	if plan.IsNull() || plan.IsUnknown() {
		return false
	}
	if state.IsNull() || state.IsUnknown() {
		return true
	}
	return plan.ValueInt64() != state.ValueInt64()
}

func changedBool(plan, state types.Bool) bool {
	if plan.IsNull() || plan.IsUnknown() {
		return false
	}
	if state.IsNull() || state.IsUnknown() {
		return true
	}
	return plan.ValueBool() != state.ValueBool()
}

// changedBoolNullable treats null as a user-meaningful value (explicit
// "clear the field"). It only skips the change-detection when the plan value
// is Unknown (i.e. the user did not set the field at all).
func changedBoolNullable(plan, state types.Bool) bool {
	if plan.IsUnknown() {
		return false
	}
	if plan.IsNull() != state.IsNull() {
		return true
	}
	if plan.IsNull() {
		return false
	}
	return plan.ValueBool() != state.ValueBool()
}

func changedIntNullable(plan, state types.Int64) bool {
	if plan.IsUnknown() {
		return false
	}
	if plan.IsNull() != state.IsNull() {
		return true
	}
	if plan.IsNull() {
		return false
	}
	return plan.ValueInt64() != state.ValueInt64()
}

func changedList(plan, state types.List) bool {
	if plan.IsUnknown() {
		return false
	}
	if plan.IsNull() != state.IsNull() {
		return true
	}
	if plan.IsNull() {
		return false
	}
	return !plan.Equal(state)
}

// listOrNull returns either nil (for a Null list, sent as JSON null to
// clear the field) or the slice of string values.
func listOrNull(l types.List) any {
	if l.IsNull() {
		return nil
	}
	out := make([]string, 0, len(l.Elements()))
	for _, e := range l.Elements() {
		if s, ok := e.(types.String); ok {
			out = append(out, s.ValueString())
		}
	}
	return out
}
