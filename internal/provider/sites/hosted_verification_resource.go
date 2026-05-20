// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
	"github.com/caputchin/terraform-provider-caputchin/internal/provider/util"
)

// NewHostedVerificationResource is the factory for the
// caputchin_hosted_verification resource.
func NewHostedVerificationResource() resource.Resource {
	return &hostedVerificationResource{}
}

type hostedVerificationResource struct {
	client *client.Client
}

type hostedVerificationModel struct {
	SiteID     types.String `tfsdk:"site_id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	WebhookURL types.String `tfsdk:"webhook_url"`
	EmailTo    types.String `tfsdk:"email_to"`
	CreatedAt  types.Int64  `tfsdk:"created_at"`
	UpdatedAt  types.Int64  `tfsdk:"updated_at"`
}

// apiHostedVerificationConfig matches the public response shape of
// GET/PUT /v1/management/hosted-verification/{siteId}. Nullable fields
// are typed as pointers so the platform's `null` decodes correctly into
// `types.StringNull()` rather than `""`.
type apiHostedVerificationConfig struct {
	SiteID     string  `json:"site_id"`
	Enabled    bool    `json:"enabled"`
	WebhookURL *string `json:"webhook_url"`
	EmailTo    *string `json:"email_to"`
	CreatedAt  *int64  `json:"created_at"`
	UpdatedAt  *int64  `json:"updated_at"`
}

func (r *hostedVerificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hosted_verification"
}

func (r *hostedVerificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Hosted-verification configuration for a site (Alpha tier or above, per ADR-0007). Singleton per `site_id`: one configuration row, upserted on every apply via PUT. Define exactly one `caputchin_hosted_verification` per `site_id`; concurrent resources targeting the same site will race on update. Destroying this resource removes Terraform tracking but does NOT disable hosted verification or clear destinations server-side; set `enabled = false` and re-apply before destroy if you want the server-side row neutralized.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "Identifier of the site this hosted-verification config belongs to. Changing this attribute forces replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether hosted verification is enabled for the site.",
				Required:    true,
			},
			"webhook_url": schema.StringAttribute{
				Description: "Customer webhook destination URL (validated as `^https?://`). May be null to leave unset.",
				Optional:    true,
				Computed:    true,
			},
			"email_to": schema.StringAttribute{
				Description: "Email destination for verification events (server-side requires `@`). May be null to leave unset.",
				Optional:    true,
				Computed:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "Creation timestamp in milliseconds since the Unix epoch.",
				Computed:    true,
			},
			"updated_at": schema.Int64Attribute{
				Description: "Last-update timestamp in milliseconds since the Unix epoch.",
				Computed:    true,
			},
		},
	}
}

func (r *hostedVerificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func hostedVerificationPath(siteID string) string {
	return "/v1/management/hosted-verification/" + siteID
}

// putAndProject sends the upsert PUT and projects the echoed config back
// into Terraform state.
func (r *hostedVerificationResource) putAndProject(ctx context.Context, plan hostedVerificationModel) (hostedVerificationModel, error) {
	body := map[string]any{
		"enabled":     plan.Enabled.ValueBool(),
		"webhook_url": nil,
		"email_to":    nil,
	}
	if !plan.WebhookURL.IsNull() && !plan.WebhookURL.IsUnknown() && plan.WebhookURL.ValueString() != "" {
		body["webhook_url"] = plan.WebhookURL.ValueString()
	}
	if !plan.EmailTo.IsNull() && !plan.EmailTo.IsUnknown() && plan.EmailTo.ValueString() != "" {
		body["email_to"] = plan.EmailTo.ValueString()
	}

	var cfg apiHostedVerificationConfig
	if err := r.client.Put(ctx, hostedVerificationPath(plan.SiteID.ValueString()), body, &cfg); err != nil {
		return hostedVerificationModel{}, err
	}
	return apiToModel(cfg), nil
}

func apiToModel(c apiHostedVerificationConfig) hostedVerificationModel {
	model := hostedVerificationModel{
		SiteID:     types.StringValue(c.SiteID),
		Enabled:    types.BoolValue(c.Enabled),
		WebhookURL: util.NullableString(c.WebhookURL),
		EmailTo:    util.NullableString(c.EmailTo),
	}
	if c.CreatedAt != nil {
		model.CreatedAt = types.Int64Value(*c.CreatedAt)
	} else {
		model.CreatedAt = types.Int64Null()
	}
	if c.UpdatedAt != nil {
		model.UpdatedAt = types.Int64Value(*c.UpdatedAt)
	} else {
		model.UpdatedAt = types.Int64Null()
	}
	return model
}

func (r *hostedVerificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hostedVerificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.putAndProject(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("hosted-verification-put-failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
}

func (r *hostedVerificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hostedVerificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var cfg apiHostedVerificationConfig
	if err := r.client.Get(ctx, hostedVerificationPath(state.SiteID.ValueString()), &cfg); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("hosted-verification-read-failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, apiToModel(cfg))...)
}

func (r *hostedVerificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan hostedVerificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.putAndProject(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("hosted-verification-put-failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
}

// Delete removes the resource from Terraform state only. The hosted-
// verification row stays in place on the server (any active webhook /
// email destinations continue receiving events); set `enabled = false`
// and re-apply before destroy if you want the row neutralized.
func (r *hostedVerificationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *hostedVerificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import id IS the site id; the resource is singleton-per-site.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), req.ID)...)
}
