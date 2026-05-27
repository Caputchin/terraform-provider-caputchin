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
	TroopID   types.String `tfsdk:"troop_id"`
	ForceGame types.Bool   `tfsdk:"force_game"`
}

// troopSecuritySettingsEnvelope matches the GET / PATCH response shape:
// `{ "troop_id": ..., "settings": { "force_game": ... } }`.
type troopSecuritySettingsEnvelope struct {
	TroopID  string                   `json:"troop_id"`
	Settings apiTroopSecuritySettings `json:"settings"`
}

type apiTroopSecuritySettings struct {
	ForceGame bool `json:"force_game"`
}

func (s apiTroopSecuritySettings) toModel(troopID string) troopSecuritySettingsModel {
	return troopSecuritySettingsModel{
		TroopID:   types.StringValue(troopID),
		ForceGame: types.BoolValue(s.ForceGame),
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
