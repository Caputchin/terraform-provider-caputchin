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

// NewListDataSource is the factory for the caputchin_sites data source,
// a plural lookup returning every site the caller can read.
func NewListDataSource() datasource.DataSource {
	return &siteListDataSource{}
}

type siteListDataSource struct {
	client *client.Client
}

type siteListItemModel struct {
	ID        types.String `tfsdk:"id"`
	Key       types.String `tfsdk:"key"`
	Name      types.String `tfsdk:"name"`
	TroopID   types.String `tfsdk:"troop_id"`
	Tier      types.String `tfsdk:"tier"`
	Disabled  types.Bool   `tfsdk:"disabled"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
}

type siteListModel struct {
	Sites []siteListItemModel `tfsdk:"sites"`
}

// apiListSite is the list-specific decode shape. The platform's
// /v1/management/sites endpoint emits TWO row shapes: the full
// publicSiteShape (caller has `read` perm) AND a names-only shape
// `{id, name, troop_id}` for memberships without read (see
// caputchin-platform/apps/web/src/app/api/v1/management/sites/route.ts
// `namesOnlySiteShape`). Without pointer fields, `encoding/json` would
// zero-fill the missing keys (key="", tier="", disabled=false,
// created_at=0) and the provider would project lies into Terraform
// state. We type the may-be-absent fields as pointers so the data-
// source Read can skip the names-only rows entirely (only sites the
// caller can actually read are useful to TF wiring).
type apiListSite struct {
	ID        string  `json:"id"`
	Key       *string `json:"key"`
	Name      string  `json:"name"`
	TroopID   string  `json:"troop_id"`
	Tier      *string `json:"tier"`
	Disabled  *bool   `json:"disabled"`
	CreatedAt *int64  `json:"created_at"`
}

type siteListEnvelope struct {
	Sites []apiListSite `json:"sites"`
}

func (d *siteListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sites"
}

func (d *siteListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Caputchin site keys the caller can read. Use this for cross-resource lookup (for example, wiring `caputchin_site_config` against existing sites). The secret is never returned by this data source; it is only available at creation time on the resource. Names-only sites (sites the caller can see exists but cannot read, per the troop-membership permission model) are intentionally excluded from this list because the additional fields would be unknown.",
		Attributes: map[string]schema.Attribute{
			"sites": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Visible sites.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true, Description: "Site identifier."},
						"key":        schema.StringAttribute{Computed: true, Description: "Public site key embedded in the widget."},
						"name":       schema.StringAttribute{Computed: true, Description: "Human-readable site name."},
						"troop_id":   schema.StringAttribute{Computed: true, Description: "Owning troop identifier."},
						"tier":       schema.StringAttribute{Computed: true, Description: "Plan tier inherited from the owning troop."},
						"disabled":   schema.BoolAttribute{Computed: true, Description: "Whether the site key is disabled."},
						"created_at": schema.Int64Attribute{Computed: true, Description: "Creation timestamp in milliseconds since the Unix epoch."},
					},
				},
			},
		},
	}
}

func (d *siteListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *siteListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var env siteListEnvelope
	if err := d.client.Get(ctx, "/v1/management/sites", &env); err != nil {
		resp.Diagnostics.AddError("sites-list-read-failed", err.Error())
		return
	}
	items := make([]siteListItemModel, 0, len(env.Sites))
	for _, s := range env.Sites {
		// Skip names-only rows (no read perm). See apiListSite.
		if s.Key == nil || s.Tier == nil || s.Disabled == nil || s.CreatedAt == nil {
			continue
		}
		items = append(items, siteListItemModel{
			ID:        types.StringValue(s.ID),
			Key:       types.StringValue(*s.Key),
			Name:      types.StringValue(s.Name),
			TroopID:   types.StringValue(s.TroopID),
			Tier:      types.StringValue(*s.Tier),
			Disabled:  types.BoolValue(*s.Disabled),
			CreatedAt: types.Int64Value(*s.CreatedAt),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, siteListModel{Sites: items})...)
}
