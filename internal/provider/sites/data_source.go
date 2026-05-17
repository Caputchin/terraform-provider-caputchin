package sites

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
	return &siteDataSource{}
}

type siteDataSource struct {
	client *client.Client
}

func (d *siteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_key"
}

func (d *siteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Caputchin site key by id. The secret is never returned by this data source; it is only available at creation time on the resource.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Required: true, Description: "Site identifier."},
			"key":        schema.StringAttribute{Computed: true, Description: "Public site key embedded in the widget."},
			"name":       schema.StringAttribute{Computed: true, Description: "Human-readable site name."},
			"team_id":    schema.StringAttribute{Computed: true, Description: "Owning team identifier."},
			"tier":       schema.StringAttribute{Computed: true, Description: "Plan tier inherited from the owning team."},
			"disabled":   schema.BoolAttribute{Computed: true, Description: "Whether the site key is disabled."},
			"created_at": schema.Int64Attribute{Computed: true, Description: "Creation timestamp in milliseconds since the Unix epoch."},
		},
	}
}

func (d *siteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *siteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// The data source intentionally omits a `secret` attribute (see Schema),
	// so we project the wire shape into a struct without the Secret column.
	var cfg struct {
		ID types.String `tfsdk:"id"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var env siteEnvelope
	if err := d.client.Get(ctx, "/v1/management/sites/"+cfg.ID.ValueString(), &env); err != nil {
		resp.Diagnostics.AddError("site-read-failed", err.Error())
		return
	}

	out := struct {
		ID        types.String `tfsdk:"id"`
		Key       types.String `tfsdk:"key"`
		Name      types.String `tfsdk:"name"`
		TeamID    types.String `tfsdk:"team_id"`
		Tier      types.String `tfsdk:"tier"`
		Disabled  types.Bool   `tfsdk:"disabled"`
		CreatedAt types.Int64  `tfsdk:"created_at"`
	}{
		ID:        types.StringValue(env.Site.ID),
		Key:       types.StringValue(env.Site.Key),
		Name:      types.StringValue(env.Site.Name),
		TeamID:    types.StringValue(env.Site.TeamID),
		Tier:      types.StringValue(env.Site.Tier),
		Disabled:  types.BoolValue(env.Site.Disabled),
		CreatedAt: types.Int64Value(env.Site.CreatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
}
