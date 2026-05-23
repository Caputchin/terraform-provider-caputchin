// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package whitelabel implements the override-pipeline resources
// (caputchin_white_label_preset, caputchin_custom_game_schema,
// caputchin_customized_game) per ADR-0061. Every resource targets exactly
// one scope: a troop (troop-wide baseline) or a site (per-site override), and
// and reaches the management API's per-axis endpoints. Game ids ride a query
// param, never the path, because they contain slashes.
package whitelabel

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// presetModel is the Terraform state/plan shape for caputchin_white_label_preset.
type presetModel struct {
	TroopID   types.String         `tfsdk:"troop_id"`
	SiteID    types.String         `tfsdk:"site_id"`
	GameID    types.String         `tfsdk:"game_id"`
	Axis      types.String         `tfsdk:"axis"`
	Name      types.String         `tfsdk:"name"`
	Values    jsontypes.Normalized `tfsdk:"values"`
	UpdatedAt types.String         `tfsdk:"updated_at"`
}

// presetWire mirrors the management API WhiteLabelPreset shape.
type presetWire struct {
	Axis      string         `json:"axis"`
	GameID    *string        `json:"game_id"`
	Name      string         `json:"name"`
	Values    map[string]any `json:"values"`
	UpdatedAt string         `json:"updated_at"`
}

type presetEnvelope struct {
	Preset presetWire `json:"preset"`
}

// schemaModel is the Terraform shape for caputchin_custom_game_schema.
type schemaModel struct {
	TroopID   types.String         `tfsdk:"troop_id"`
	SiteID    types.String         `tfsdk:"site_id"`
	GameID    types.String         `tfsdk:"game_id"`
	Axis      types.String         `tfsdk:"axis"`
	Schema    jsontypes.Normalized `tfsdk:"schema"`
	UpdatedAt types.String         `tfsdk:"updated_at"`
}

type schemaWire struct {
	Axis      string         `json:"axis"`
	Schema    map[string]any `json:"schema"`
	UpdatedAt string         `json:"updated_at"`
}

type schemaEnvelope struct {
	Schema schemaWire `json:"schema"`
}

// gameModel is the Terraform shape for caputchin_customized_game (registry row).
type gameModel struct {
	TroopID types.String `tfsdk:"troop_id"`
	SiteID  types.String `tfsdk:"site_id"`
	GameID  types.String `tfsdk:"game_id"`
	Source  types.String `tfsdk:"source"`
}

type gameWire struct {
	GameID string `json:"game_id"`
	Source string `json:"source"`
}

type gameEnvelope struct {
	Game gameWire `json:"game"`
}
