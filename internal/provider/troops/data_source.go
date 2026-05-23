// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewDataSource is the factory consumed by the provider's DataSources() list.
func NewDataSource() datasource.DataSource {
	return &troopDataSource{}
}

type troopDataSource struct {
	client *client.Client
}

func (d *troopDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_troop"
}

func (d *troopDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Caputchin troop by id.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Required: true, Description: "Troop identifier."},
			"name":        schema.StringAttribute{Computed: true, Description: "Troop name."},
			"account_id":  schema.StringAttribute{Computed: true, Description: "Owning account identifier."},
			"kind":        schema.StringAttribute{Computed: true, Description: "Troop kind (`personal` or `shared`)."},
			"tier":        schema.StringAttribute{Computed: true, Description: "Plan tier inherited from the owning account."},
			"created_at":  schema.Int64Attribute{Computed: true, Description: "Creation timestamp in milliseconds since the Unix epoch."},
			"can_manage":  schema.BoolAttribute{Computed: true, Description: "Whether the calling principal can manage this troop (owning account/account-PAT, or the principal's membership `manage` permission)."},
			"owner_email": schema.StringAttribute{Computed: true, Description: "Email of the troop's owning account. Returned only to callers who manage the troop; null otherwise."},
		},
	}
}

func (d *troopDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *troopDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg troopModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var env troopEnvelope
	if err := d.client.Get(ctx, "/v1/management/troops/"+cfg.ID.ValueString(), &env); err != nil {
		resp.Diagnostics.AddError("troop-read-failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, env.Troop.toModel())...)
}
