package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AutomationsDataSource{}

type AutomationsDataSource struct {
	client *client.Client
}

func NewAutomationsDataSource() datasource.DataSource {
	return &AutomationsDataSource{}
}

// AutomationModel is shared between the resource and data source.
type AutomationModel struct {
	ID          types.String `tfsdk:"id"`
	Alias       types.String `tfsdk:"alias"`
	Description types.String `tfsdk:"description"`
	Mode        types.String `tfsdk:"mode"`
	Trigger     types.String `tfsdk:"trigger"`
	Condition   types.String `tfsdk:"condition"`
	Action      types.String `tfsdk:"action"`
}

type AutomationsDataSourceModel struct {
	Automations []AutomationModel `tfsdk:"automations"`
}

func (d *AutomationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automations"
}

func (d *AutomationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all automations from Home Assistant.",
		Attributes: map[string]schema.Attribute{
			"automations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true, Description: "Automation ID."},
						"alias":       schema.StringAttribute{Computed: true, Description: "Display name."},
						"description": schema.StringAttribute{Computed: true, Description: "Description."},
						"mode":        schema.StringAttribute{Computed: true, Description: "Automation mode."},
						"trigger":     schema.StringAttribute{Computed: true, Description: "JSON-encoded triggers."},
						"condition":   schema.StringAttribute{Computed: true, Description: "JSON-encoded conditions."},
						"action":      schema.StringAttribute{Computed: true, Description: "JSON-encoded actions."},
					},
				},
			},
		},
	}
}

func (d *AutomationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AutomationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AutomationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	automations, err := d.client.GetAutomations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch automations", err.Error())
		return
	}

	state.Automations = make([]AutomationModel, len(automations))
	for i, a := range automations {
		state.Automations[i] = automationToModel(&a)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
