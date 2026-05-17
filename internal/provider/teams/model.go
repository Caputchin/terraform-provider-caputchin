// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package teams implements the caputchin_team resource and data source.
package teams

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// teamModel is the Terraform state shape for caputchin_team. The wire shape
// (apiTeam) is decoded separately because Terraform's types.String /
// types.Int64 don't round-trip through encoding/json without bespoke
// MarshalJSON plumbing.
type teamModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	AccountID types.String `tfsdk:"account_id"`
	Kind      types.String `tfsdk:"kind"`
	Tier      types.String `tfsdk:"tier"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
}

// teamEnvelope is the {team: {...}} envelope every team route returns.
type teamEnvelope struct {
	Team apiTeam `json:"team"`
}

// apiTeam is the raw wire shape (no Terraform tag indirection) used purely
// for JSON decoding. teamModel uses Terraform types and cannot be unmarshalled
// directly without custom UnmarshalJSON plumbing — keeping them separate is
// cheaper.
type apiTeam struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AccountID string `json:"account_id"`
	Kind      string `json:"kind"`
	Tier      string `json:"tier"`
	CreatedAt int64  `json:"created_at"`
}

func (t apiTeam) toModel() teamModel {
	return teamModel{
		ID:        types.StringValue(t.ID),
		Name:      types.StringValue(t.Name),
		AccountID: types.StringValue(t.AccountID),
		Kind:      types.StringValue(t.Kind),
		Tier:      types.StringValue(t.Tier),
		CreatedAt: types.Int64Value(t.CreatedAt),
	}
}
