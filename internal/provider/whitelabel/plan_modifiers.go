// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dropEmptyLeaves is a plan modifier for a JSON-object string attribute that
// strips top-level leaves whose value is null or an empty string, mirroring
// the management API's server-side cleanValues (it persists neither). Without
// it a `values` carrying an empty/null leaf diffs forever: the server drops
// the leaf, the refreshed state lacks it, but the config still declares it, so
// every plan shows a phantom update. Stripping the same leaves at plan time
// keeps plan == state == what the API stores.
//
// Leaves are kept verbatim (json.RawMessage) so a kept value's exact bytes -
// including number precision - survive untouched; only top-level key order is
// normalized by re-marshal, which jsontypes.Normalized compares away anyway.
type dropEmptyLeaves struct{}

func dropEmptyLeavesModifier() planmodifier.String { return dropEmptyLeaves{} }

func (dropEmptyLeaves) Description(_ context.Context) string {
	return "drops null and empty-string leaves to match the API's persisted shape (prevents a perpetual diff)"
}

func (m dropEmptyLeaves) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (dropEmptyLeaves) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(req.PlanValue.ValueString()), &obj); err != nil {
		// Not a JSON object (or invalid) - leave it untouched; the
		// jsontypes.Normalized type's own validation surfaces the error.
		return
	}
	changed := false
	for k, raw := range obj {
		if s := string(raw); s == "null" || s == `""` {
			delete(obj, k)
			changed = true
		}
	}
	if !changed {
		return
	}
	cleaned, err := json.Marshal(obj)
	if err != nil {
		return
	}
	resp.PlanValue = types.StringValue(string(cleaned))
}
