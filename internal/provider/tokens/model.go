// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package tokens implements the caputchin_account_token resource: minting
// and revoking management tokens (`account` and `troop` types).
// Attachment of troop-PATs to specific troops happens via the
// caputchin_troop_pat resource in the `troops` package.
package tokens

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tokenModel is the Terraform state shape for caputchin_account_token.
// SecretVersion is a provider-tracked rotation counter (not echoed by the
// API) that drives in-place credential rotation. Bumping
// the value in a plan fires POST /v1/management/tokens/{id}/rotate on
// the next apply; the token row's id and name stay stable, the prefix
// rotates together with the secret half, and the rotated values land in
// the existing Secret and Prefix attributes.
type tokenModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Prefix        types.String `tfsdk:"prefix"`
	Secret        types.String `tfsdk:"secret"`
	CreatedAt     types.Int64  `tfsdk:"created_at"`
	SecretVersion types.Int64  `tfsdk:"secret_version"`
}

// createEnvelope matches the POST /tokens response: { token: {...} }.
type createEnvelope struct {
	Token apiTokenWithValue `json:"token"`
}

// listEnvelope matches the GET /tokens response: { tokens: [{...}] }. Read
// uses this to find the row by id, since the API does not expose a
// GET-by-id endpoint.
type listEnvelope struct {
	Tokens []apiToken `json:"tokens"`
}

// apiTokenWithValue is the wire shape returned only at POST time. `value`
// is the full bearer secret, present once.
type apiTokenWithValue struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Prefix    string `json:"prefix"`
	Value     string `json:"value"`
	CreatedAt int64  `json:"created_at"`
}

// apiToken is the wire shape returned by GET /tokens (no secret).
type apiToken struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Prefix     string `json:"prefix"`
	LastUsedAt *int64 `json:"last_used_at"`
	CreatedAt  int64  `json:"created_at"`
}

// rotateEnvelope matches the POST /tokens/{id}/rotate response: { token,
// prefix } where token is the replacement bearer-token string returned
// ONCE and prefix is the rotated 16-char leading prefix (same as
// token.slice(0, 16)). Both rotate together; the token row's
// id and name are unchanged across rotation.
type rotateEnvelope struct {
	Token  string `json:"token"`
	Prefix string `json:"prefix"`
}
