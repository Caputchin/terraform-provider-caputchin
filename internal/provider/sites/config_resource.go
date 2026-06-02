// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewConfigResource is the factory consumed by the provider's Resources() list.
func NewConfigResource() resource.Resource {
	return &siteConfigResource{}
}

type siteConfigResource struct {
	client *client.Client
}

func (r *siteConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_config"
}

func (r *siteConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Per-site configuration for a Caputchin site key. Singleton: one configuration row per site. Fields you do not set retain the server-side defaults derived from the site's plan tier.\n\nDefine exactly one `caputchin_site_config` per `site_id`; concurrent resources targeting the same site will race on update.\n\nDestroying this resource removes Terraform tracking but does NOT reset the server-side configuration; values set via this resource remain in effect until explicitly changed.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "Identifier of the site this configuration belongs to. Changing this attribute forces replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pow_difficulty": schema.Int64Attribute{
				Description: "Proof-of-work difficulty (1-8). Higher values make the challenge harder to solve.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 8),
				},
			},
			"pow_challenge_count": schema.Int64Attribute{
				Description: "Number of proof-of-work sub-challenges per session (1-500).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 500),
				},
			},
			"instrumentation": schema.BoolAttribute{
				Description: "Whether the browser-instrumentation challenge runs (default true). Set false to remove the `'unsafe-eval'` Content-Security-Policy requirement from pages embedding the widget, at the cost of automated-browser detection (proof-of-work and game replay still run). `obfuscation_level` and `block_automated_browsers` only take effect while this is true.",
				Optional:    true,
				Computed:    true,
			},
			"obfuscation_level": schema.Int64Attribute{
				Description: "Browser-instrumentation obfuscation level (1-10). Only effective while `instrumentation` is true.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"block_automated_browsers": schema.BoolAttribute{
				Description: "If true, reject sessions identified as automated browsers (Selenium, Puppeteer, etc.).",
				Optional:    true,
				Computed:    true,
			},
			"block_non_browser_ua": schema.BoolAttribute{
				Description: "If true, reject sessions whose User-Agent does not look like a browser. May be null to leave unset.",
				Optional:    true,
				Computed:    true,
			},
			"required_headers": schema.ListAttribute{
				Description: "Headers that must be present on incoming requests. May be null to leave unset.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"ratelimit_max": schema.Int64Attribute{
				Description: "Maximum verification requests per second (1-10000), capped at the plan tier ceiling. Values above the tier ceiling are rejected by the server. May be null to use the server's plan-tier default.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 10000),
				},
			},
			"cors_origins": schema.ListAttribute{
				Description: "Allowed origins for verification calls (e.g. `[\"https://example.com\"]`). May be null to leave unset (no CORS check).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *siteConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected-provider-data",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *siteConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildPatchBody(plan, siteConfigModel{})

	if len(body) > 0 {
		if err := r.client.Patch(ctx, configPath(plan.SiteID.ValueString()), body, nil); err != nil {
			resp.Diagnostics.AddError("site-config-create-failed", err.Error())
			return
		}
	}

	// Always refresh from the server so Computed defaults the user did not set
	// land in state with their actual values.
	r.refreshState(ctx, plan.SiteID, &resp.Diagnostics, func(m siteConfigModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

func (r *siteConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.refreshState(ctx, state.SiteID, &resp.Diagnostics, func(m siteConfigModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

func (r *siteConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state siteConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildPatchBody(plan, state)

	if len(body) > 0 {
		if err := r.client.Patch(ctx, configPath(plan.SiteID.ValueString()), body, nil); err != nil {
			resp.Diagnostics.AddError("site-config-update-failed", err.Error())
			return
		}
	}

	r.refreshState(ctx, plan.SiteID, &resp.Diagnostics, func(m siteConfigModel) {
		resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
	})
}

// Delete removes the resource from Terraform state only. There is no
// server-side delete for a singleton config (the site keeps whatever values
// the config currently holds, including defaults).
func (r *siteConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *siteConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import id IS the site id (the resource is singleton-per-site).
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), req.ID)...)
}

func configPath(siteID string) string {
	return "/v1/management/sites/" + siteID + "/cap-config"
}

func (r *siteConfigResource) refreshState(ctx context.Context, siteIDTF types.String, diags diagSink, sink func(siteConfigModel)) {
	siteID := siteIDTF.ValueString()
	var env siteConfigEnvelope
	if err := r.client.Get(ctx, configPath(siteID), &env); err != nil {
		diags.AddError("site-config-read-failed", err.Error())
		return
	}
	sink(env.Config.toModel(ctx, siteID, diags))
}

// buildPatchBody assembles the PATCH body containing only the fields whose
// plan value differs from the prior state value. On Create, state is the
// zero-value siteConfigModel; every plan-set field is treated as a change.
func (r *siteConfigResource) buildPatchBody(plan, state siteConfigModel) map[string]any {
	body := map[string]any{}

	if changedInt(plan.PowDifficulty, state.PowDifficulty) {
		body["difficulty"] = plan.PowDifficulty.ValueInt64()
	}
	if changedInt(plan.PowChallengeCount, state.PowChallengeCount) {
		body["challenge_count"] = plan.PowChallengeCount.ValueInt64()
	}
	if changedBool(plan.Instrumentation, state.Instrumentation) {
		body["instrumentation"] = plan.Instrumentation.ValueBool()
	}
	if changedInt(plan.ObfuscationLevel, state.ObfuscationLevel) {
		body["obfuscation_level"] = plan.ObfuscationLevel.ValueInt64()
	}
	if changedBool(plan.BlockAutomatedBrowsers, state.BlockAutomatedBrowsers) {
		body["block_automated_browsers"] = plan.BlockAutomatedBrowsers.ValueBool()
	}
	if changedBoolNullable(plan.BlockNonBrowserUA, state.BlockNonBrowserUA) {
		if plan.BlockNonBrowserUA.IsNull() {
			body["block_non_browser_ua"] = nil
		} else {
			body["block_non_browser_ua"] = plan.BlockNonBrowserUA.ValueBool()
		}
	}
	if changedIntNullable(plan.RatelimitMax, state.RatelimitMax) {
		if plan.RatelimitMax.IsNull() {
			body["ratelimit_max"] = nil
		} else {
			body["ratelimit_max"] = plan.RatelimitMax.ValueInt64()
		}
	}
	if changedList(plan.RequiredHeaders, state.RequiredHeaders) {
		body["required_headers"] = listOrNull(plan.RequiredHeaders)
	}
	if changedList(plan.CorsOrigins, state.CorsOrigins) {
		body["cors_origins"] = listOrNull(plan.CorsOrigins)
	}
	return body
}

// diagSink is the subset of diag.Diagnostics this file needs.
type diagSink interface {
	AddError(summary, detail string)
}
