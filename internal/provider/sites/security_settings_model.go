// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// siteSecuritySettingsModel is the Terraform state shape for
// caputchin_site_security_settings (the per-site game gate). Singleton per site.
type siteSecuritySettingsModel struct {
	SiteID      types.String `tfsdk:"site_id"`
	RequireGame types.Bool   `tfsdk:"require_game"`
}

// siteSecuritySettingsEnvelope matches the GET / PATCH response shape:
// `{ "site_id": ..., "settings": { "require_game": ... } }`.
type siteSecuritySettingsEnvelope struct {
	SiteID   string                  `json:"site_id"`
	Settings apiSiteSecuritySettings `json:"settings"`
}

type apiSiteSecuritySettings struct {
	RequireGame bool `json:"require_game"`
}

func (s apiSiteSecuritySettings) toModel(siteID string) siteSecuritySettingsModel {
	return siteSecuritySettingsModel{
		SiteID:      types.StringValue(siteID),
		RequireGame: types.BoolValue(s.RequireGame),
	}
}
