package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ScenesDataSource{}

type ScenesDataSource struct {
	client *client.Client
}

func NewScenesDataSource() datasource.DataSource {
	return &ScenesDataSource{}
}

type SceneSummaryModel struct {
	ID       types.String `tfsdk:"id"`
	EntityID types.String `tfsdk:"entity_id"`
	Name     types.String `tfsdk:"name"`
	Entities types.List   `tfsdk:"entities"`
}

type ScenesDataSourceModel struct {
	Scenes []SceneSummaryModel `tfsdk:"scenes"`
}

func (d *ScenesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scenes"
}

func (d *ScenesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the scenes loaded in Home Assistant (derived from entity states).",
		Attributes: map[string]schema.Attribute{
			"scenes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":        schema.StringAttribute{Computed: true, Description: "Scene ID, if available."},
						"entity_id": schema.StringAttribute{Computed: true, Description: "The scene entity ID (e.g. scene.movie_time)."},
						"name":      schema.StringAttribute{Computed: true, Description: "Friendly name of the scene."},
						"entities": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Entity IDs controlled by this scene.",
						},
					},
				},
			},
		},
	}
}

func (d *ScenesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *ScenesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ScenesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scenes, err := d.client.GetScenes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch scenes", err.Error())
		return
	}

	state.Scenes = make([]SceneSummaryModel, len(scenes))
	for i, s := range scenes {
		entities, diags := types.ListValueFrom(ctx, types.StringType, s.Entities)
		resp.Diagnostics.Append(diags...)
		state.Scenes[i] = SceneSummaryModel{
			ID:       types.StringValue(s.ID),
			EntityID: types.StringValue(s.EntityID),
			Name:     types.StringValue(s.Name),
			Entities: entities,
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
