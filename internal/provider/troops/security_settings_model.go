// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// troopSecuritySettingsModel is the Terraform state shape for
// caputchin_troop_security_settings (the troop-wide game-gate ceiling).
// Singleton per troop.
type troopSecuritySettingsModel struct {
	TroopID       types.String `tfsdk:"troop_id"`
	ForceGame     types.Bool   `tfsdk:"force_game"`
	PreviewMode   types.Bool   `tfsdk:"preview_mode"`
	Reuse         types.Bool   `tfsdk:"reuse"`
	ReuseWindowMs types.Int64  `tfsdk:"reuse_window_ms"`
	ReusePersist  types.Bool   `tfsdk:"reuse_persist"`
}

// troopSecuritySettingsEnvelope matches the GET / PATCH response shape:
// `{ "troop_id": ..., "settings": { "force_game": ..., "preview_mode": ...,
// "reuse": ..., "reuse_window_ms": ..., "reuse_persist": ... } }`.
type troopSecuritySettingsEnvelope struct {
	TroopID  string                   `json:"troop_id"`
	Settings apiTroopSecuritySettings `json:"settings"`
}

// apiTroopSecuritySettings is the wire shape. The verification-reuse trio is the
// troop-wide default a site key inherits (default+override, resolved
// site ?? troop ?? default): null on the troop row = no default, so they are
// pointers to preserve the null-vs-false distinction.
type apiTroopSecuritySettings struct {
	ForceGame     bool   `json:"force_game"`
	PreviewMode   bool   `json:"preview_mode"`
	Reuse         *bool  `json:"reuse"`
	ReuseWindowMs *int64 `json:"reuse_window_ms"`
	ReusePersist  *bool  `json:"reuse_persist"`
}

func (s apiTroopSecuritySettings) toModel(troopID string) troopSecuritySettingsModel {
	return troopSecuritySettingsModel{
		TroopID:       types.StringValue(troopID),
		ForceGame:     types.BoolValue(s.ForceGame),
		PreviewMode:   types.BoolValue(s.PreviewMode),
		Reuse:         nullableBool(s.Reuse),
		ReuseWindowMs: nullableInt64(s.ReuseWindowMs),
		ReusePersist:  nullableBool(s.ReusePersist),
	}
}

// changedBool reports whether the planned bool differs from prior state. Unknown
// plan (user did not set the field) is never a change; an unset prior state with
// a set plan is.
func changedBool(plan, state types.Bool) bool {
	if plan.IsNull() || plan.IsUnknown() {
		return false
	}
	if state.IsNull() || state.IsUnknown() {
		return true
	}
	return plan.ValueBool() != state.ValueBool()
}

// nullableBool / nullableInt64 map a wire pointer to a framework value, keeping
// null distinct from false/0 (a default+override field inherits the parent when
// null). changedBoolNullable / changedIntNullable treat null as a user-meaningful
// value (explicit "clear the default"), only skipping when the plan is Unknown.
// Mirrors the site package's config_conversion helpers.
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
