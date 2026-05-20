// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package account implements the caputchin_account data source.
package account

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewDataSource is the factory consumed by the provider's DataSources() list.
func NewDataSource() datasource.DataSource {
	return &accountDataSource{}
}

type accountDataSource struct {
	client *client.Client
}

type accountModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
}

type accountEnvelope struct {
	Account apiAccount `json:"account"`
}

type apiAccount struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
}

func (d *accountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *accountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Metadata for the Caputchin account that owns the provider's management token.\n\nIntended for account-level credentials (session cookie or account-PAT). Per ADR-0027, once troop-PAT typing ships in the platform, troop-PAT callers will be rejected with an upstream authorization error; configurations that may run under a troop-PAT should not depend on this data source.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true, Description: "Account identifier."},
			"email":      schema.StringAttribute{Computed: true, Description: "Account email address."},
			"created_at": schema.Int64Attribute{Computed: true, Description: "Creation timestamp in milliseconds since the Unix epoch."},
		},
	}
}

func (d *accountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *accountDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var env accountEnvelope
	if err := d.client.Get(ctx, "/v1/management/me/account", &env); err != nil {
		resp.Diagnostics.AddError("account-read-failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, accountModel{
		ID:        types.StringValue(env.Account.ID),
		Email:     types.StringValue(env.Account.Email),
		CreatedAt: types.Int64Value(env.Account.CreatedAt),
	})...)
}
