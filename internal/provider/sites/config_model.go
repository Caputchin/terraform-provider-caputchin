// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// siteConfigModel is the Terraform state shape for caputchin_site_config.
// Two HCL attribute names diverge from the wire shape: the proof-of-work
// knobs are prefixed `pow_` in HCL to signal what they tune. The mapping
// happens in apiSiteConfig.toModel() and siteConfigModel.toAPIPatch().
type siteConfigModel struct {
	SiteID                 types.String `tfsdk:"site_id"`
	PowDifficulty          types.Int64  `tfsdk:"pow_difficulty"`
	PowChallengeCount      types.Int64  `tfsdk:"pow_challenge_count"`
	Instrumentation        types.Bool   `tfsdk:"instrumentation"`
	ObfuscationLevel       types.Int64  `tfsdk:"obfuscation_level"`
	BlockAutomatedBrowsers types.Bool   `tfsdk:"block_automated_browsers"`
	BlockNonBrowserUA      types.Bool   `tfsdk:"block_non_browser_ua"`
	RequiredHeaders        types.List   `tfsdk:"required_headers"`
	RatelimitMax           types.Int64  `tfsdk:"ratelimit_max"`
	CorsOrigins            types.List   `tfsdk:"cors_origins"`
}

// siteConfigEnvelope matches the GET / PATCH response shape.
type siteConfigEnvelope struct {
	SiteID string        `json:"site_id"`
	Config apiSiteConfig `json:"config"`
}

// apiSiteConfig is the wire shape returned by the management API.
// `BlockNonBrowserUA`, `RequiredHeaders`, `RatelimitMax`, and `CorsOrigins`
// are nullable on the wire (pointers / explicit nulls); represented here as
// pointers so the JSON decoder can distinguish absent / null / value.
type apiSiteConfig struct {
	Instrumentation        bool     `json:"instrumentation"`
	Difficulty             int64    `json:"difficulty"`
	ChallengeCount         int64    `json:"challenge_count"`
	ObfuscationLevel       int64    `json:"obfuscation_level"`
	BlockAutomatedBrowsers bool     `json:"block_automated_browsers"`
	BlockNonBrowserUA      *bool    `json:"block_non_browser_ua"`
	RequiredHeaders        []string `json:"required_headers"`
	RatelimitMax           *int64   `json:"ratelimit_max"`
	CorsOrigins            []string `json:"cors_origins"`
}
