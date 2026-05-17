package sites

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// int64UseStateForUnknown mirrors the helper in the teams package; the
// framework lacks a stock int64 implementation.
type int64UseStateForUnknown struct{}

func (m int64UseStateForUnknown) Description(_ context.Context) string {
	return "When the state value is known, use it as the planned value to avoid spurious diffs."
}

func (m int64UseStateForUnknown) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m int64UseStateForUnknown) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.State.Raw.IsNull() {
		return
	}
	if !req.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}
