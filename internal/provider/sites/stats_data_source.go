// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewStatsDataSource is the factory consumed by the provider's DataSources() list.
func NewStatsDataSource() datasource.DataSource {
	return &statsDataSource{}
}

type statsDataSource struct {
	client *client.Client
}

type siteStatsModel struct {
	SiteID                   types.String `tfsdk:"site_id"`
	SessionsStarted          types.Int64  `tfsdk:"sessions_started"`
	SessionsClientCompleted  types.Int64  `tfsdk:"sessions_client_completed"`
	SessionsServerVerified   types.Int64  `tfsdk:"sessions_server_verified"`
	FailedClientCompletion   types.Int64  `tfsdk:"failed_client_completion"`
	FailedServerVerification types.Int64  `tfsdk:"failed_server_verification"`
	RateLimitRejections      types.Int64  `tfsdk:"rate_limit_rejections"`
	ChallengeBlocked         types.Int64  `tfsdk:"challenge_blocked"`
}

type apiSiteStats struct {
	SiteID                   string `json:"site_id"`
	SessionsStarted          int64  `json:"sessions_started"`
	SessionsClientCompleted  int64  `json:"sessions_client_completed"`
	SessionsServerVerified   int64  `json:"sessions_server_verified"`
	FailedClientCompletion   int64  `json:"failed_client_completion"`
	FailedServerVerification int64  `json:"failed_server_verification"`
	RateLimitRejections      int64  `json:"rate_limit_rejections"`
	ChallengeBlocked         int64  `json:"challenge_blocked"`
}

func (d *statsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_stats"
}

func (d *statsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lifetime counters for a Caputchin site key. Returns monotonic totals across the site's history; time-series breakdowns are not exposed via this data source.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Required:    true,
				Description: "Site identifier.",
			},
			"sessions_started": schema.Int64Attribute{
				Computed:    true,
				Description: "Total sessions started by the widget.",
			},
			"sessions_client_completed": schema.Int64Attribute{
				Computed:    true,
				Description: "Sessions in which the browser-side challenge was completed.",
			},
			"sessions_server_verified": schema.Int64Attribute{
				Computed:    true,
				Description: "Sessions whose token was successfully verified by `/siteverify`.",
			},
			"failed_client_completion": schema.Int64Attribute{
				Computed:    true,
				Description: "Sessions in which the client-side completion attempt was rejected.",
			},
			"failed_server_verification": schema.Int64Attribute{
				Computed:    true,
				Description: "Sessions in which the server-side verification call was rejected.",
			},
			"rate_limit_rejections": schema.Int64Attribute{
				Computed:    true,
				Description: "Issuance attempts rejected by the per-second rate limit.",
			},
			"challenge_blocked": schema.Int64Attribute{
				Computed:    true,
				Description: "Issuance attempts blocked by the per-site security filters.",
			},
		},
	}
}

func (d *statsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = c
}

func (d *statsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg struct {
		SiteID types.String `tfsdk:"site_id"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stats apiSiteStats
	if err := d.client.Get(ctx, "/v1/management/sites/"+cfg.SiteID.ValueString()+"/stats", &stats); err != nil {
		resp.Diagnostics.AddError("site-stats-read-failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, siteStatsModel{
		SiteID:                   types.StringValue(stats.SiteID),
		SessionsStarted:          types.Int64Value(stats.SessionsStarted),
		SessionsClientCompleted:  types.Int64Value(stats.SessionsClientCompleted),
		SessionsServerVerified:   types.Int64Value(stats.SessionsServerVerified),
		FailedClientCompletion:   types.Int64Value(stats.FailedClientCompletion),
		FailedServerVerification: types.Int64Value(stats.FailedServerVerification),
		RateLimitRejections:      types.Int64Value(stats.RateLimitRejections),
		ChallengeBlocked:         types.Int64Value(stats.ChallengeBlocked),
	})...)
}
