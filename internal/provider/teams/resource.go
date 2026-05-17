package teams

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// NewResource is the factory consumed by the provider's Resources() list.
func NewResource() resource.Resource {
	return &teamResource{}
}

type teamResource struct {
	client *client.Client
}

func (r *teamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *teamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Caputchin team — the tenant boundary that owns site keys. Only shared teams are creatable; personal teams are auto-created per account and cannot be managed by this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Server-issued team identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable team name. Modifiable in place.",
				Required:    true,
			},
			"account_id": schema.StringAttribute{
				Description: "Owning account identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kind": schema.StringAttribute{
				Description: "Team kind. Always `shared` for resources managed by this provider.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tier": schema.StringAttribute{
				Description: "Plan tier inherited from the owning account. Read-only.",
				Computed:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "Creation timestamp in milliseconds since the Unix epoch.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64UseStateForUnknown{},
				},
			},
		},
	}
}

func (r *teamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{"name": plan.Name.ValueString()}
	var env teamEnvelope
	if err := r.client.Post(ctx, "/v1/management/teams", body, &env); err != nil {
		resp.Diagnostics.AddError("team-create-failed", err.Error())
		return
	}

	if env.Team.Kind == "personal" {
		// Defensive — the route only creates shared teams, but if a future
		// API change ever returns a personal team here we want the surprise
		// surfaced loudly rather than silently stored in state.
		resp.Diagnostics.AddError(
			"unexpected-personal-team",
			"The management API returned a personal team for a create call. Personal teams cannot be managed by this provider.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, env.Team.toModel())...)
}

func (r *teamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var env teamEnvelope
	err := r.client.Get(ctx, "/v1/management/teams/"+state.ID.ValueString(), &env)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("team-read-failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, env.Team.toModel())...)
}

func (r *teamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{"name": plan.Name.ValueString()}
	var env teamEnvelope
	if err := r.client.Patch(ctx, "/v1/management/teams/"+plan.ID.ValueString(), body, &env); err != nil {
		resp.Diagnostics.AddError("team-update-failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, env.Team.toModel())...)
}

func (r *teamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, "/v1/management/teams/"+state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("team-delete-failed", err.Error())
	}
}

func (r *teamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
