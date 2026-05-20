// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewListDataSource is the factory for the caputchin_troops data source,
// a plural lookup returning every troop visible to the caller.
func NewListDataSource() datasource.DataSource {
	return &troopListDataSource{}
}

type troopListDataSource struct {
	client *client.Client
}

type troopListItemModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	AccountID types.String `tfsdk:"account_id"`
	Kind      types.String `tfsdk:"kind"`
	Tier      types.String `tfsdk:"tier"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
}

type troopListModel struct {
	Troops []troopListItemModel `tfsdk:"troops"`
}

type troopListEnvelope struct {
	Troops []apiTroop `json:"troops"`
}

func (d *troopListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_troops"
}

func (d *troopListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "All Caputchin troops visible to the caller (own-account troops plus troops where the caller has a membership). Use this for cross-resource lookup.",
		Attributes: map[string]schema.Attribute{
			"troops": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Visible troops.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true, Description: "Troop identifier."},
						"name":       schema.StringAttribute{Computed: true, Description: "Troop name."},
						"account_id": schema.StringAttribute{Computed: true, Description: "Owning account identifier."},
						"kind":       schema.StringAttribute{Computed: true, Description: "Troop kind (`personal` or `shared`)."},
						"tier":       schema.StringAttribute{Computed: true, Description: "Plan tier inherited from the owning account."},
						"created_at": schema.Int64Attribute{Computed: true, Description: "Creation timestamp in milliseconds since the Unix epoch."},
					},
				},
			},
		},
	}
}

func (d *troopListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *troopListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var env troopListEnvelope
	if err := d.client.Get(ctx, "/v1/management/troops", &env); err != nil {
		resp.Diagnostics.AddError("troops-list-read-failed", err.Error())
		return
	}
	items := make([]troopListItemModel, 0, len(env.Troops))
	for _, t := range env.Troops {
		items = append(items, troopListItemModel{
			ID:        types.StringValue(t.ID),
			Name:      types.StringValue(t.Name),
			AccountID: types.StringValue(t.AccountID),
			Kind:      types.StringValue(t.Kind),
			Tier:      types.StringValue(t.Tier),
			CreatedAt: types.Int64Value(t.CreatedAt),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, troopListModel{Troops: items})...)
}
