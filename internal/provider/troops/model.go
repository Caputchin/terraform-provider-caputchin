// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package troops implements the caputchin_troop resource and data source.
package troops

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// troopModel is the Terraform state shape for caputchin_troop. The wire shape
// (apiTroop) is decoded separately because Terraform's types.String /
// types.Int64 don't round-trip through encoding/json without bespoke
// MarshalJSON plumbing.
type troopModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	AccountID  types.String `tfsdk:"account_id"`
	Kind       types.String `tfsdk:"kind"`
	Tier       types.String `tfsdk:"tier"`
	CreatedAt  types.Int64  `tfsdk:"created_at"`
	CanManage  types.Bool   `tfsdk:"can_manage"`
	OwnerEmail types.String `tfsdk:"owner_email"`
}

// troopEnvelope is the {troop: {...}} envelope every troop route returns.
type troopEnvelope struct {
	Troop apiTroop `json:"troop"`
}

// apiTroop is the raw wire shape (no Terraform tag indirection) used purely
// for JSON decoding. troopModel uses Terraform types and cannot be unmarshalled
// directly without custom UnmarshalJSON plumbing; keeping them separate is
// cheaper.
type apiTroop struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	AccountID  string  `json:"account_id"`
	Kind       string  `json:"kind"`
	Tier       string  `json:"tier"`
	CreatedAt  int64   `json:"created_at"`
	CanManage  bool    `json:"can_manage"`
	OwnerEmail *string `json:"owner_email"`
}

// ownerEmailValue maps the nullable wire field to a Terraform string: the API
// returns owner_email only to managers (null otherwise), so non-managers get
// a null attribute rather than an empty string.
func ownerEmailValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func (t apiTroop) toModel() troopModel {
	return troopModel{
		ID:         types.StringValue(t.ID),
		Name:       types.StringValue(t.Name),
		AccountID:  types.StringValue(t.AccountID),
		Kind:       types.StringValue(t.Kind),
		Tier:       types.StringValue(t.Tier),
		CreatedAt:  types.Int64Value(t.CreatedAt),
		CanManage:  types.BoolValue(t.CanManage),
		OwnerEmail: ownerEmailValue(t.OwnerEmail),
	}
}
