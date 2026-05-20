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
// shape (apiSite) is decoded separately; see troops/model.go for the same
// pattern and rationale. SecretVersion and RotationTriggers are
// provider-side knobs (not echoed by the API) that drive in-place
// rotation and full-replacement rotation respectively per ADR-0051.
type siteModel struct {
	ID               types.String `tfsdk:"id"`
	Key              types.String `tfsdk:"key"`
	Name             types.String `tfsdk:"name"`
	TroopID          types.String `tfsdk:"troop_id"`
	Tier             types.String `tfsdk:"tier"`
	Disabled         types.Bool   `tfsdk:"disabled"`
	CreatedAt        types.Int64  `tfsdk:"created_at"`
	Secret           types.String `tfsdk:"secret"`
	SecretVersion    types.Int64  `tfsdk:"secret_version"`
	RotationTriggers types.Map    `tfsdk:"rotation_triggers"`
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
	TroopID   string `json:"troop_id"`
	Tier      string `json:"tier"`
	Disabled  bool   `json:"disabled"`
	CreatedAt int64  `json:"created_at"`
}

// toModel projects the API shape into Terraform state. The caller supplies
// the secret (it lives outside the wire shape, present only on Create
// and rotate-secret then preserved across reads from prior state), the
// secret_version (provider-tracked counter that drives rotation), and
// the rotation_triggers map (user-supplied replace signal).
func (s apiSite) toModel(secret types.String, secretVersion types.Int64, rotationTriggers types.Map) siteModel {
	return siteModel{
		ID:               types.StringValue(s.ID),
		Key:              types.StringValue(s.Key),
		Name:             types.StringValue(s.Name),
		TroopID:          types.StringValue(s.TroopID),
		Tier:             types.StringValue(s.Tier),
		Disabled:         types.BoolValue(s.Disabled),
		CreatedAt:        types.Int64Value(s.CreatedAt),
		Secret:           secret,
		SecretVersion:    secretVersion,
		RotationTriggers: rotationTriggers,
	}
}
