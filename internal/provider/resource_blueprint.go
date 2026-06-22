package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &BlueprintResource{}

type BlueprintResource struct {
	client *client.Client
}

func NewBlueprintResource() resource.Resource {
	return &BlueprintResource{}
}

type BlueprintResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Domain    types.String `tfsdk:"domain"`
	Path      types.String `tfsdk:"path"`
	SourceURL types.String `tfsdk:"source_url"`
	Blueprint types.String `tfsdk:"blueprint"`
}

func (r *BlueprintResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint"
}

func (r *BlueprintResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Home Assistant blueprint. The blueprint YAML is managed by Terraform; " +
			"Home Assistant does not expose the stored YAML for reading, so changes made outside Terraform " +
			"to the YAML content are not detected (only deletion of the blueprint is).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier in the form \"<domain>/<path>\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "Blueprint domain (e.g. \"automation\", \"script\", or \"template\").",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Relative path of the blueprint file within the domain (e.g. \"my_dir/my_blueprint.yaml\").",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_url": schema.StringAttribute{
				Optional:    true,
				Description: "Optional URL the blueprint was imported from.",
			},
			"blueprint": schema.StringAttribute{
				Required:    true,
				Description: "The blueprint YAML content.",
			},
		},
	}
}

func (r *BlueprintResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *BlueprintResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BlueprintResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SaveBlueprint(ctx, modelToBlueprint(plan), false); err != nil {
		resp.Diagnostics.AddError("Failed to create blueprint", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/" + plan.Path.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *BlueprintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BlueprintResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta, err := r.client.GetBlueprint(ctx, state.Domain.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read blueprint", err.Error())
		return
	}
	if meta == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Home Assistant does not return the stored YAML, so the blueprint content
	// and source_url are kept as-is in state; only existence is refreshed here.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *BlueprintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BlueprintResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SaveBlueprint(ctx, modelToBlueprint(plan), true); err != nil {
		resp.Diagnostics.AddError("Failed to update blueprint", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/" + plan.Path.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *BlueprintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BlueprintResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteBlueprint(ctx, state.Domain.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete blueprint", err.Error())
	}
}

func modelToBlueprint(m BlueprintResourceModel) client.Blueprint {
	return client.Blueprint{
		Domain:    m.Domain.ValueString(),
		Path:      m.Path.ValueString(),
		YAML:      m.Blueprint.ValueString(),
		SourceURL: valueOrNil(m.SourceURL),
	}
}
