// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// siteSecuritySettingsModel is the Terraform state shape for
// caputchin_site_security_settings (the per-site game gate). Singleton per site.
type siteSecuritySettingsModel struct {
	SiteID        types.String `tfsdk:"site_id"`
	RequireGame   types.Bool   `tfsdk:"require_game"`
	PreviewMode   types.Bool   `tfsdk:"preview_mode"`
	Reuse         types.Bool   `tfsdk:"reuse"`
	ReuseWindowMs types.Int64  `tfsdk:"reuse_window_ms"`
	ReusePersist  types.Bool   `tfsdk:"reuse_persist"`
}

// siteSecuritySettingsEnvelope matches the GET / PATCH response shape:
// `{ "site_id": ..., "settings": { "require_game": ..., "preview_mode": ...,
// "reuse": ..., "reuse_window_ms": ..., "reuse_persist": ... } }`.
type siteSecuritySettingsEnvelope struct {
	SiteID   string                  `json:"site_id"`
	Settings apiSiteSecuritySettings `json:"settings"`
}

// apiSiteSecuritySettings is the wire shape returned by the management API.
// PreviewMode is nullable on the wire (null inherits the troop default,
// resolved as site ?? troop ?? false); ReuseWindowMs is nullable too (absent/null
// falls back to the server's default window). Both are pointers here.
type apiSiteSecuritySettings struct {
	RequireGame   bool   `json:"require_game"`
	PreviewMode   *bool  `json:"preview_mode"`
	Reuse         bool   `json:"reuse"`
	ReuseWindowMs *int64 `json:"reuse_window_ms"`
	ReusePersist  bool   `json:"reuse_persist"`
}

func (s apiSiteSecuritySettings) toModel(siteID string) siteSecuritySettingsModel {
	return siteSecuritySettingsModel{
		SiteID:        types.StringValue(siteID),
		RequireGame:   types.BoolValue(s.RequireGame),
		PreviewMode:   nullableBool(s.PreviewMode),
		Reuse:         types.BoolValue(s.Reuse),
		ReuseWindowMs: nullableInt64(s.ReuseWindowMs),
		ReusePersist:  types.BoolValue(s.ReusePersist),
	}
}
