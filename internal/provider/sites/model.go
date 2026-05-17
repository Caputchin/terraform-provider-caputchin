// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package sites implements the caputchin_site_key resource and related data
// sources. The package will grow to host caputchin_site_config and
// caputchin_site_stats in later phases.
package sites

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// siteModel is the Terraform state shape for caputchin_site_key. The wire
// shape (apiSite) is decoded separately; see teams/model.go for the same
// pattern and rationale.
type siteModel struct {
	ID        types.String `tfsdk:"id"`
	Key       types.String `tfsdk:"key"`
	Name      types.String `tfsdk:"name"`
	TeamID    types.String `tfsdk:"team_id"`
	Tier      types.String `tfsdk:"tier"`
	Disabled  types.Bool   `tfsdk:"disabled"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	Secret    types.String `tfsdk:"secret"`
}

// siteEnvelope matches GET / PATCH / POST 201 responses.
type siteEnvelope struct {
	Site   apiSite `json:"site"`
	Secret string  `json:"secret,omitempty"` // present only on Create
}

type apiSite struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	TeamID    string `json:"team_id"`
	Tier      string `json:"tier"`
	Disabled  bool   `json:"disabled"`
	CreatedAt int64  `json:"created_at"`
}

// toModel projects the API shape into Terraform state. The caller supplies
// the secret (it lives outside the wire shape — present only on Create, then
// preserved across reads from prior state).
func (s apiSite) toModel(secret types.String) siteModel {
	return siteModel{
		ID:        types.StringValue(s.ID),
		Key:       types.StringValue(s.Key),
		Name:      types.StringValue(s.Name),
		TeamID:    types.StringValue(s.TeamID),
		Tier:      types.StringValue(s.Tier),
		Disabled:  types.BoolValue(s.Disabled),
		CreatedAt: types.Int64Value(s.CreatedAt),
		Secret:    secret,
	}
}
