// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package whitelabel implements the override-pipeline resources
// (caputchin_white_label_preset, caputchin_custom_game_schema,
// caputchin_customized_game). Every resource targets exactly
// one scope (a troop troop-wide baseline, or a site per-site override) and
// reaches the management API's per-axis endpoints. Game ids ride a query
// param, never the path, because they contain slashes.
package whitelabel

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// presetModel is the Terraform state/plan shape for caputchin_white_label_preset.
type presetModel struct {
	ID        types.String         `tfsdk:"id"`
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
	ID        types.String         `tfsdk:"id"`
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
	ID              types.String `tfsdk:"id"`
	TroopID         types.String `tfsdk:"troop_id"`
	SiteID          types.String `tfsdk:"site_id"`
	GameID          types.String `tfsdk:"game_id"`
	Source          types.String `tfsdk:"source"`
	AutoUpdate      types.Bool   `tfsdk:"auto_update"`
	PinnedVersion   types.String `tfsdk:"pinned_version"`
	UpdateAvailable types.Bool   `tfsdk:"update_available"`
}

type gameWire struct {
	GameID          string  `json:"game_id"`
	Source          string  `json:"source"`
	AutoUpdate      bool    `json:"auto_update"`
	PinnedVersionID *string `json:"pinned_version_id"`
	UpdateAvailable bool    `json:"update_available"`
}

type gameEnvelope struct {
	Game gameWire `json:"game"`
}

// ---------- synthetic resource ids ----------
//
// These resources have a composite natural key (scope + scope id + game +
// axis + name), so they carry a computed `id` that encodes it in the same
// pipe-delimited form ImportState parses. Terraform needs a single id for
// import + state tracking, and it lets users reference `.id`.

// idScope resolves the scope segment (kind + value) of a resource id from the
// mutually-exclusive troop_id / site_id pair (one is always set; site wins).
func idScope(troopID, siteID types.String) (kind, value string) {
	if siteID.ValueString() != "" {
		return "site", siteID.ValueString()
	}
	return "troop", troopID.ValueString()
}

// buildPresetID: scope|id|game|axis|name (game empty for widget-shell).
func buildPresetID(m presetModel) string {
	kind, value := idScope(m.TroopID, m.SiteID)
	return fmt.Sprintf("%s|%s|%s|%s|%s", kind, value, m.GameID.ValueString(), m.Axis.ValueString(), m.Name.ValueString())
}

// buildSchemaID: scope|id|game|axis.
func buildSchemaID(m schemaModel) string {
	kind, value := idScope(m.TroopID, m.SiteID)
	return fmt.Sprintf("%s|%s|%s|%s", kind, value, m.GameID.ValueString(), m.Axis.ValueString())
}

// buildGameID: scope|id|game.
func buildGameID(m gameModel) string {
	kind, value := idScope(m.TroopID, m.SiteID)
	return fmt.Sprintf("%s|%s|%s", kind, value, m.GameID.ValueString())
}

// runArtifactModel is the Terraform shape for caputchin_custom_game_run_artifact.
type runArtifactModel struct {
	ID           types.String `tfsdk:"id"`
	TroopID      types.String `tfsdk:"troop_id"`
	SiteID       types.String `tfsdk:"site_id"`
	GameID       types.String `tfsdk:"game_id"`
	RunPath      types.String `tfsdk:"run_path"`
	ModulePaths  types.List   `tfsdk:"module_paths"`
	SourceHash   types.String `tfsdk:"source_hash"`
	VersionHash  types.String `tfsdk:"version_hash"`
	SelfCheckOK  types.Bool   `tfsdk:"self_check_ok"`
	UploadedAt   types.String `tfsdk:"uploaded_at"`
}

// runArtifactFileWire mirrors the api-schemas RunArtifactFile shape.
type runArtifactFileWire struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Integrity string `json:"integrity"`
	Size      *int64 `json:"size"`
}

// runArtifactDetailWire mirrors RunArtifactDetail. The detail endpoint returns
// this body directly (no envelope) so the resource unmarshals into the wire
// type at the top level.
type runArtifactDetailWire struct {
	VersionHash string                `json:"version_hash"`
	UploadedAt  string                `json:"uploaded_at"`
	SelfCheckOK bool                  `json:"self_check_ok"`
	Run         runArtifactFileWire   `json:"run"`
	Modules     []runArtifactFileWire `json:"modules"`
}

// runArtifactUploadResponseWire mirrors the UploadRunArtifactResponse schema.
type runArtifactUploadResponseWire struct {
	VersionHash string `json:"version_hash"`
	SelfCheckOK bool   `json:"self_check_ok"`
	Idempotent  bool   `json:"idempotent"`
}

// buildRunArtifactID: scope|id|game.
func buildRunArtifactID(m runArtifactModel) string {
	kind, value := idScope(m.TroopID, m.SiteID)
	return fmt.Sprintf("%s|%s|%s", kind, value, m.GameID.ValueString())
}
