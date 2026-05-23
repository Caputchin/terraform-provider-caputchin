// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// isSet reports whether a Terraform string attribute carries a real,
// non-empty value (not null/unknown).
func isSet(s types.String) bool {
	return !s.IsNull() && !s.IsUnknown() && s.ValueString() != ""
}

// scopeSegment maps the resource's scope attributes to the management API
// path segment + id. Site scope wins when site_id is set; otherwise troop.
// ExactlyOneOf config validation guarantees exactly one is set.
func scopeSegment(troopID, siteID types.String) (string, string) {
	if isSet(siteID) {
		return "sites", siteID.ValueString()
	}
	return "troops", troopID.ValueString()
}

// presetPath builds the per-axis preset endpoint. Widget-shell (no game)
// lives under white-label/{axis}; a game-axis preset under
// game-customization/{axis} with the game id as a query param.
func presetPath(troopID, siteID, gameID types.String, axis, name string) string {
	scope, id := scopeSegment(troopID, siteID)
	q := url.Values{}
	q.Set("name", name)
	if isSet(gameID) {
		q.Set("game", gameID.ValueString())
		return fmt.Sprintf("/v1/management/%s/%s/game-customization/%s/preset?%s", scope, id, axis, q.Encode())
	}
	return fmt.Sprintf("/v1/management/%s/%s/white-label/%s/preset?%s", scope, id, axis, q.Encode())
}

// schemaPath builds the per-axis custom-game schema endpoint (game-axis only).
func schemaPath(troopID, siteID, gameID types.String, axis string) string {
	scope, id := scopeSegment(troopID, siteID)
	q := url.Values{}
	q.Set("game", gameID.ValueString())
	return fmt.Sprintf("/v1/management/%s/%s/game-customization/%s/schema?%s", scope, id, axis, q.Encode())
}

// gamesPath builds the customized-games collection endpoint (register/list).
func gamesPath(troopID, siteID types.String) string {
	scope, id := scopeSegment(troopID, siteID)
	return fmt.Sprintf("/v1/management/%s/%s/game-customization/games", scope, id)
}

// gamePath builds the single customized-game endpoint (get/delete-cascade).
func gamePath(troopID, siteID, gameID types.String) string {
	scope, id := scopeSegment(troopID, siteID)
	q := url.Values{}
	q.Set("game", gameID.ValueString())
	return fmt.Sprintf("/v1/management/%s/%s/game-customization/game?%s", scope, id, q.Encode())
}
