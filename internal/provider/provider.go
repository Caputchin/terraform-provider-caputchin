// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/account"
	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
	"github.com/caputchin/terraform-provider-caputchin/internal/provider/sites"
	"github.com/caputchin/terraform-provider-caputchin/internal/provider/tokens"
	"github.com/caputchin/terraform-provider-caputchin/internal/provider/troops"
	"github.com/caputchin/terraform-provider-caputchin/internal/provider/whitelabel"
)

const (
	defaultEndpoint = "https://caputchin.com/api"
	envEndpoint     = "CAPUTCHIN_ENDPOINT"
	envToken        = "CAPUTCHIN_MANAGEMENT_TOKEN"
)

type caputchinProvider struct {
	version string
}

type caputchinProviderModel struct {
	Endpoint        types.String `tfsdk:"endpoint"`
	ManagementToken types.String `tfsdk:"management_token"`
}

// New returns the provider factory.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &caputchinProvider{version: version}
	}
}

func (p *caputchinProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "caputchin"
	resp.Version = p.version
}

func (p *caputchinProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Caputchin troops, site keys, per-site configuration, and read account / stats data.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "Caputchin management API endpoint. Defaults to https://caputchin.com/api. May be overridden via the CAPUTCHIN_ENDPOINT environment variable.",
				Optional:    true,
			},
			"management_token": schema.StringAttribute{
				Description: "Caputchin management API token (account-PAT or troop-PAT). May be supplied via the CAPUTCHIN_MANAGEMENT_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *caputchinProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data caputchinProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := strOrEnv(data.Endpoint, envEndpoint, defaultEndpoint)
	token := strOrEnv(data.ManagementToken, envToken, "")

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("management_token"),
			"missing-management-token",
			"A Caputchin management token is required. Set the `management_token` provider argument or the CAPUTCHIN_MANAGEMENT_TOKEN environment variable.",
		)
		return
	}

	c := client.NewClient(endpoint, token, p.version)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *caputchinProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		troops.NewResource,
		troops.NewMemberResource,
		troops.NewPatResource,
		sites.NewResource,
		sites.NewConfigResource,
		sites.NewHostedVerificationResource,
		tokens.NewResource,
		whitelabel.NewPresetResource,
		whitelabel.NewSchemaResource,
		whitelabel.NewGameResource,
	}
}

func (p *caputchinProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		troops.NewDataSource,
		troops.NewListDataSource,
		sites.NewDataSource,
		sites.NewListDataSource,
		sites.NewStatsDataSource,
		account.NewDataSource,
	}
}

func strOrEnv(v types.String, envKey, fallback string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	if env := os.Getenv(envKey); env != "" {
		return env
	}
	return fallback
}
