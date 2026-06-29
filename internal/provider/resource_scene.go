package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SceneResource{}

type SceneResource struct {
	client *client.Client
}

type SceneModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Icon     types.String `tfsdk:"icon"`
	Entities types.String `tfsdk:"entities"`
	AreaID   types.String `tfsdk:"area_id"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Visible  types.Bool   `tfsdk:"visible"`
}

func NewSceneResource() resource.Resource {
	return &SceneResource{}
}

func (r *SceneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scene"
}

func (r *SceneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Home Assistant scene.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name":     schema.StringAttribute{Required: true, Description: "Display name of the scene."},
			"icon":     schema.StringAttribute{Optional: true, Computed: true, Description: "Icon for the scene (e.g. \"mdi:flower\")."},
			"entities": schema.StringAttribute{Required: true, Description: "JSON-encoded object mapping entity IDs to the state they should take when the scene is activated."},
			"area_id":  schema.StringAttribute{Optional: true, Description: "Area to assign this scene to."},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the scene is enabled. When false, sets disabled_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"visible": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the scene is visible. When false, sets hidden_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SceneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SceneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SceneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := modelToScene(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scene config", err.Error())
		return
	}

	created, err := r.client.CreateScene(ctx, *s)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create scene", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, sceneToModel(created))...)
}

func (r *SceneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SceneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scene, err := r.client.GetScene(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read scene", err.Error())
		return
	}
	if scene == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, sceneToModel(scene))...)
}

func (r *SceneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SceneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SceneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := modelToScene(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scene config", err.Error())
		return
	}
	s.ID = state.ID.ValueString()

	updated, err := r.client.UpdateScene(ctx, *s)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update scene", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, sceneToModel(updated))...)
}

func (r *SceneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SceneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteScene(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete scene", err.Error())
	}
}

func sceneToModel(s *client.Scene) SceneModel {
	return SceneModel{
		ID:       types.StringValue(s.ID),
		Name:     types.StringValue(s.Name),
		Icon:     types.StringValue(s.Icon),
		Entities: types.StringValue(rawToString(s.Entities, "{}")),
		AreaID:   stringPtrValue(s.AreaID),
		Enabled:  boolPtrValue(s.Enabled),
		Visible:  boolPtrValue(s.Visible),
	}
}

func modelToScene(m SceneModel) (*client.Scene, error) {
	entities, err := parseRawJSON(m.Entities.ValueString(), "entities")
	if err != nil {
		return nil, err
	}
	return &client.Scene{
		Name:     m.Name.ValueString(),
		Icon:     m.Icon.ValueString(),
		Entities: entities,
		AreaID:   valueOrNil(m.AreaID),
		Enabled:  boolValueOrNil(m.Enabled),
		Visible:  boolValueOrNil(m.Visible),
	}, nil
}
