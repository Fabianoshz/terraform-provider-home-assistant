package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AreasDataSource{}

type AreasDataSource struct {
	client *client.Client
}

func NewAreasDataSource() datasource.DataSource {
	return &AreasDataSource{}
}

type AreasDataSourceModel struct {
	Areas []AreaModel `tfsdk:"areas"`
}

type AreaModel struct {
	AreaID  types.String `tfsdk:"area_id"`
	Name    types.String `tfsdk:"name"`
	Icon    types.String `tfsdk:"icon"`
	FloorID types.String `tfsdk:"floor_id"`
}

func (d *AreasDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_areas"
}

func (d *AreasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all areas from the Home Assistant area registry.",
		Attributes: map[string]schema.Attribute{
			"areas": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"area_id":  schema.StringAttribute{Computed: true, Description: "Area ID."},
						"name":     schema.StringAttribute{Computed: true, Description: "Display name of the area."},
						"icon":     schema.StringAttribute{Computed: true, Optional: true, Description: "MDI icon for the area."},
						"floor_id": schema.StringAttribute{Computed: true, Optional: true, Description: "Floor this area belongs to."},
					},
				},
			},
		},
	}
}

func (d *AreasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AreasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AreasDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	areas, err := d.client.GetAreas(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch areas", err.Error())
		return
	}

	state.Areas = make([]AreaModel, len(areas))
	for i, a := range areas {
		state.Areas[i] = AreaModel{
			AreaID:  types.StringValue(a.AreaID),
			Name:    types.StringValue(a.Name),
			Icon:    stringPtrValue(a.Icon),
			FloorID: stringPtrValue(a.FloorID),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
