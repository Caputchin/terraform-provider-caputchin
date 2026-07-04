// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// siteSecuritySettingsModel is the Terraform state shape for
// caputchin_site_security_settings (the per-site game gate). Singleton per site.
type siteSecuritySettingsModel struct {
	SiteID          types.String `tfsdk:"site_id"`
	RequireGame     types.Bool   `tfsdk:"require_game"`
	PreviewMode     types.Bool   `tfsdk:"preview_mode"`
	Reuse           types.Bool   `tfsdk:"reuse"`
	ReuseWindowMs   types.Int64  `tfsdk:"reuse_window_ms"`
	ReusePersist    types.Bool   `tfsdk:"reuse_persist"`
	ProxyGate       types.Bool   `tfsdk:"proxy_gate"`
	ProxyTtlSeconds types.Int64  `tfsdk:"proxy_ttl_seconds"`
	ProxyFailMode   types.String `tfsdk:"proxy_fail_mode"`
}

// siteSecuritySettingsEnvelope matches the GET / PATCH response shape:
// `{ "site_id": ..., "settings": { "require_game": ..., "preview_mode": ...,
// "reuse": ..., "reuse_window_ms": ..., "reuse_persist": ... } }`.
type siteSecuritySettingsEnvelope struct {
	SiteID   string                  `json:"site_id"`
	Settings apiSiteSecuritySettings `json:"settings"`
}

// apiSiteSecuritySettings is the wire shape returned by the management API.
// Verification reuse (`reuse` / `reuse_window_ms` / `reuse_persist`) and
// `preview_mode` are default+override fields: null on a site row inherits the
// troop default (resolved site ?? troop ?? default), so they are pointers here to
// preserve the null-vs-false distinction. `reuse_window_ms` / `proxy_ttl_seconds`
// null fall back to the server's clamped default.
type apiSiteSecuritySettings struct {
	RequireGame     bool    `json:"require_game"`
	PreviewMode     *bool   `json:"preview_mode"`
	Reuse           *bool   `json:"reuse"`
	ReuseWindowMs   *int64  `json:"reuse_window_ms"`
	ReusePersist    *bool   `json:"reuse_persist"`
	ProxyGate       bool    `json:"proxy_gate"`
	ProxyTtlSeconds *int64  `json:"proxy_ttl_seconds"`
	ProxyFailMode   *string `json:"proxy_fail_mode"`
}

func (s apiSiteSecuritySettings) toModel(siteID string) siteSecuritySettingsModel {
	return siteSecuritySettingsModel{
		SiteID:          types.StringValue(siteID),
		RequireGame:     types.BoolValue(s.RequireGame),
		PreviewMode:     nullableBool(s.PreviewMode),
		Reuse:           nullableBool(s.Reuse),
		ReuseWindowMs:   nullableInt64(s.ReuseWindowMs),
		ReusePersist:    nullableBool(s.ReusePersist),
		ProxyGate:       types.BoolValue(s.ProxyGate),
		ProxyTtlSeconds: nullableInt64(s.ProxyTtlSeconds),
		ProxyFailMode:   nullableString(s.ProxyFailMode),
	}
}
