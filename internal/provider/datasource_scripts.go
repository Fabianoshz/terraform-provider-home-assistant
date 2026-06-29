package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ScriptsDataSource{}

type ScriptsDataSource struct {
	client *client.Client
}

func NewScriptsDataSource() datasource.DataSource {
	return &ScriptsDataSource{}
}

type ScriptSummaryModel struct {
	ID       types.String `tfsdk:"id"`
	EntityID types.String `tfsdk:"entity_id"`
	Name     types.String `tfsdk:"name"`
	Mode     types.String `tfsdk:"mode"`
	State    types.String `tfsdk:"state"`
}

type ScriptsDataSourceModel struct {
	Scripts []ScriptSummaryModel `tfsdk:"scripts"`
}

func (d *ScriptsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scripts"
}

func (d *ScriptsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the scripts loaded in Home Assistant (derived from entity states).",
		Attributes: map[string]schema.Attribute{
			"scripts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":        schema.StringAttribute{Computed: true, Description: "Script object ID (the suffix of the entity ID)."},
						"entity_id": schema.StringAttribute{Computed: true, Description: "The script entity ID (e.g. script.notify_everyone)."},
						"name":      schema.StringAttribute{Computed: true, Description: "Friendly name of the script."},
						"mode":      schema.StringAttribute{Computed: true, Description: "Script mode: single, restart, queued, or parallel."},
						"state":     schema.StringAttribute{Computed: true, Description: "Current state (\"on\" while running, otherwise \"off\")."},
					},
				},
			},
		},
	}
}

func (d *ScriptsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScriptsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ScriptsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scripts, err := d.client.GetScripts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch scripts", err.Error())
		return
	}

	state.Scripts = make([]ScriptSummaryModel, len(scripts))
	for i, s := range scripts {
		state.Scripts[i] = ScriptSummaryModel{
			ID:       types.StringValue(s.ID),
			EntityID: types.StringValue(s.EntityID),
			Name:     types.StringValue(s.Name),
			Mode:     types.StringValue(s.Mode),
			State:    types.StringValue(s.State),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
