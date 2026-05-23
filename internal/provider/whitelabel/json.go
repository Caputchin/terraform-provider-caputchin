// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
)

// jsonToMap decodes a normalized-JSON Terraform attribute into a generic
// map for the request body. Null/unknown decodes to an empty object.
func jsonToMap(n jsontypes.Normalized) (map[string]any, error) {
	if n.IsNull() || n.IsUnknown() {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(n.ValueString()), &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// mapToNormalized encodes the server's returned object back into a
// normalized-JSON attribute. jsontypes.Normalized compares semantically, so
// key order / whitespace differences from the server never produce a diff.
func mapToNormalized(m map[string]any) (jsontypes.Normalized, error) {
	if m == nil {
		m = map[string]any{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return jsontypes.NewNormalizedNull(), err
	}
	return jsontypes.NewNormalizedValue(string(b)), nil
}
