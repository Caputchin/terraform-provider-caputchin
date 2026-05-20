// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package util holds cross-resource helpers that don't belong to any
// single domain package. Members are tiny on purpose: anything bigger
// belongs in its owning domain package (sites, troops, tokens, etc.).
package util

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NullableString projects a `*string` (the shape produced by
// encoding/json for a JSON field tagged optional/nullable) into a
// Terraform String. A nil pointer maps to `types.StringNull()`; a
// non-nil pointer maps to `types.StringValue`.
func NullableString(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}
