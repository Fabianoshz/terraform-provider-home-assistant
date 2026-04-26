package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &EntitiesDataSource{}

type EntitiesDataSource struct {
	client *client.Client
}

func NewEntitiesDataSource() datasource.DataSource {
	return &EntitiesDataSource{}
}

type EntitiesDataSourceModel struct {
	Domain   types.String  `tfsdk:"domain"`
	Entities []EntityStateModel `tfsdk:"entities"`
}

type EntityStateModel struct {
	EntityID    types.String `tfsdk:"entity_id"`
	State       types.String `tfsdk:"state"`
	Attributes  types.String `tfsdk:"attributes"`
	LastChanged types.String `tfsdk:"last_changed"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func (d *EntitiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entities"
}

func (d *EntitiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches Home Assistant entity states, optionally filtered by domain (e.g. light, switch, sensor).",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Description: "Optional domain filter (e.g. \"light\", \"sensor\"). Leave empty to return all entities.",
				Optional:    true,
			},
			"entities": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"entity_id":    schema.StringAttribute{Computed: true, Description: "The entity ID (e.g. light.living_room)."},
						"state":        schema.StringAttribute{Computed: true, Description: "Current state of the entity."},
						"attributes":   schema.StringAttribute{Computed: true, Description: "JSON-encoded map of entity attributes."},
						"last_changed": schema.StringAttribute{Computed: true, Description: "Timestamp when the state last changed."},
						"last_updated": schema.StringAttribute{Computed: true, Description: "Timestamp when the entity was last updated."},
					},
				},
			},
		},
	}
}

func (d *EntitiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EntitiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state EntitiesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entities, err := d.client.GetStates(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch entity states", err.Error())
		return
	}

	domain := state.Domain.ValueString()
	var result []EntityStateModel
	for _, e := range entities {
		if domain != "" && domainFromEntityID(e.EntityID) != domain {
			continue
		}
		attrsJSON, err := json.Marshal(e.Attributes)
		if err != nil {
			resp.Diagnostics.AddError("Failed to encode attributes", err.Error())
			return
		}
		result = append(result, EntityStateModel{
			EntityID:    types.StringValue(e.EntityID),
			State:       types.StringValue(e.State),
			Attributes:  types.StringValue(string(attrsJSON)),
			LastChanged: types.StringValue(e.LastChanged),
			LastUpdated: types.StringValue(e.LastUpdated),
		})
	}
	if result == nil {
		result = []EntityStateModel{}
	}
	state.Entities = result
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func domainFromEntityID(entityID string) string {
	for i, c := range entityID {
		if c == '.' {
			return entityID[:i]
		}
	}
	return entityID
}
