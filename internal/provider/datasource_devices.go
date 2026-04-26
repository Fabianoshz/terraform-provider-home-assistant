package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DevicesDataSource{}

type DevicesDataSource struct {
	client *client.Client
}

func NewDevicesDataSource() datasource.DataSource {
	return &DevicesDataSource{}
}

type FilterModel struct {
	Source types.String `tfsdk:"source"`
	Type   types.String `tfsdk:"type"`
	Value  types.String `tfsdk:"value"`
}

type ConnectionModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

type EntityModel struct {
	EntityID     types.String `tfsdk:"entity_id"`
	Name         types.String `tfsdk:"name"`
	OriginalName types.String `tfsdk:"original_name"`
	Platform     types.String `tfsdk:"platform"`
	AreaID       types.String `tfsdk:"area_id"`
	DisabledBy   types.String `tfsdk:"disabled_by"`
	Disabled     types.Bool   `tfsdk:"disabled"`
	Icon         types.String `tfsdk:"icon"`
}

type DeviceModel struct {
	ID           types.String      `tfsdk:"id"`
	Name         types.String      `tfsdk:"name"`
	NameByUser   types.String      `tfsdk:"name_by_user"`
	Manufacturer types.String      `tfsdk:"manufacturer"`
	Model        types.String      `tfsdk:"model"`
	AreaID       types.String      `tfsdk:"area_id"`
	HWVersion    types.String      `tfsdk:"hw_version"`
	SerialNumber types.String      `tfsdk:"serial_number"`
	DisabledBy   types.String      `tfsdk:"disabled_by"`
	Connections  []ConnectionModel `tfsdk:"connections"`
	Identifiers  []ConnectionModel `tfsdk:"identifiers"`
	Entities     []EntityModel     `tfsdk:"entities"`
}

type DevicesDataSourceModel struct {
	Filter                *FilterModel  `tfsdk:"filter"`
	HideDisabledEntities  types.Bool    `tfsdk:"hide_disabled_entities"`
	Devices               []DeviceModel `tfsdk:"devices"`
}

func (d *DevicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_devices"
}

func (d *DevicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	connectionAttrs := map[string]schema.Attribute{
		"type":  schema.StringAttribute{Computed: true, Description: "Connection/identifier type (e.g. \"mac\", \"zha\")."},
		"value": schema.StringAttribute{Computed: true, Description: "Connection/identifier value."},
	}

	resp.Schema = schema.Schema{
		Description: "Fetches devices from the Home Assistant device registry, optionally filtered.",
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				Description: "Filter devices by a connection or identifier.",
				Attributes: map[string]schema.Attribute{
					"source": schema.StringAttribute{
						Optional:    true,
						Description: "Which field to search: \"connections\" or \"identifiers\".",
					},
					"type": schema.StringAttribute{
						Optional:    true,
						Description: "The type to match (e.g. \"mac\" for connections, \"zha\" for identifiers).",
					},
					"value": schema.StringAttribute{
						Optional:    true,
						Description: "The value to match.",
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"hide_disabled_entities": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, entities with a disabled_by value are excluded from each device's entity list.",
			},
			"devices": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true, Description: "Device ID."},
						"name":          schema.StringAttribute{Computed: true, Description: "Device name."},
						"name_by_user":  schema.StringAttribute{Computed: true, Optional: true, Description: "User-defined name override."},
						"manufacturer":  schema.StringAttribute{Computed: true, Optional: true, Description: "Device manufacturer."},
						"model":         schema.StringAttribute{Computed: true, Optional: true, Description: "Device model."},
						"area_id":       schema.StringAttribute{Computed: true, Optional: true, Description: "Area the device belongs to."},
						"hw_version":    schema.StringAttribute{Computed: true, Optional: true, Description: "Hardware version."},
						"serial_number": schema.StringAttribute{Computed: true, Optional: true, Description: "Serial number."},
						"disabled_by":   schema.StringAttribute{Computed: true, Optional: true, Description: "What disabled this device, if anything."},
						"connections":   schema.ListNestedAttribute{Computed: true, Description: "Physical connections (e.g. mac, upnp).", NestedObject: schema.NestedAttributeObject{Attributes: connectionAttrs}},
						"identifiers": schema.ListNestedAttribute{Computed: true, Description: "Integration-specific identifiers (e.g. zha, mqtt, hue).", NestedObject: schema.NestedAttributeObject{Attributes: connectionAttrs}},
						"entities": schema.ListNestedAttribute{
							Computed:    true,
							Description: "Entities registered under this device.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"entity_id":     schema.StringAttribute{Computed: true, Description: "Entity ID (e.g. sensor.living_room_temperature)."},
									"name":          schema.StringAttribute{Computed: true, Optional: true, Description: "User-defined name override in HA."},
									"original_name": schema.StringAttribute{Computed: true, Optional: true, Description: "Name provided by the integration (e.g. ESPHome friendly name)."},
									"platform":      schema.StringAttribute{Computed: true, Description: "Integration platform (e.g. esphome, hue, zha)."},
									"area_id":       schema.StringAttribute{Computed: true, Optional: true, Description: "Area override for this entity."},
									"disabled_by": schema.StringAttribute{Computed: true, Optional: true, Description: "What disabled this entity, if anything."},
									"disabled":    schema.BoolAttribute{Computed: true, Description: "True if the entity is disabled."},
									"icon":        schema.StringAttribute{Computed: true, Optional: true, Description: "Custom icon."},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *DevicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DevicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DevicesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Filter != nil {
		source := config.Filter.Source.ValueString()
		if source != "connections" && source != "identifiers" {
			resp.Diagnostics.AddError("Invalid filter.source", "filter.source must be \"connections\" or \"identifiers\".")
			return
		}
		if config.Filter.Type.IsNull() || config.Filter.Type.ValueString() == "" {
			resp.Diagnostics.AddError("Missing filter.type", "filter.type is required when filter block is set.")
			return
		}
		if config.Filter.Value.IsNull() || config.Filter.Value.ValueString() == "" {
			resp.Diagnostics.AddError("Missing filter.value", "filter.value is required when filter block is set.")
			return
		}
	}

	devices, entityEntries, err := d.client.GetDevicesAndEntities(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch devices", err.Error())
		return
	}

	hideDisabled := config.HideDisabledEntities.ValueBool()

	entitiesByDevice := make(map[string][]EntityModel)
	for _, e := range entityEntries {
		if e.DeviceID == nil {
			continue
		}
		if hideDisabled && e.DisabledBy != nil {
			continue
		}
		entitiesByDevice[*e.DeviceID] = append(entitiesByDevice[*e.DeviceID], EntityModel{
			EntityID:     types.StringValue(e.EntityID),
			Name:         stringPtrValue(e.Name),
			OriginalName: stringPtrValue(e.OriginalName),
			Platform:     types.StringValue(e.Platform),
			AreaID:       stringPtrValue(e.AreaID),
			DisabledBy:   stringPtrValue(e.DisabledBy),
			Disabled:     types.BoolValue(e.DisabledBy != nil),
			Icon:         stringPtrValue(e.Icon),
		})
	}

	var result []DeviceModel
	for _, dev := range devices {
		if config.Filter != nil && !matchesFilter(dev, config.Filter) {
			continue
		}
		entities := entitiesByDevice[dev.ID]
		if entities == nil {
			entities = []EntityModel{}
		}
		result = append(result, DeviceModel{
			ID:           types.StringValue(dev.ID),
			Name:         types.StringValue(dev.Name),
			NameByUser:   stringPtrValue(dev.NameByUser),
			Manufacturer: stringPtrValue(dev.Manufacturer),
			Model:        stringPtrValue(dev.Model),
			AreaID:       stringPtrValue(dev.AreaID),
			HWVersion:    stringPtrValue(dev.HWVersion),
			SerialNumber: stringPtrValue(dev.SerialNumber),
			DisabledBy:   stringPtrValue(dev.DisabledBy),
			Connections:  pairsToConnections(dev.Connections),
			Identifiers:  pairsToConnections(dev.Identifiers),
			Entities:     entities,
		})
	}

	if result == nil {
		result = []DeviceModel{}
	}

	config.Devices = result
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func matchesFilter(dev client.Device, f *FilterModel) bool {
	typ := f.Type.ValueString()
	val := f.Value.ValueString()

	var pairs [][]string
	switch f.Source.ValueString() {
	case "connections":
		pairs = dev.Connections
	case "identifiers":
		pairs = dev.Identifiers
	}

	for _, pair := range pairs {
		if len(pair) == 2 && pair[0] == typ && pair[1] == val {
			return true
		}
	}
	return false
}

func pairsToConnections(pairs [][]string) []ConnectionModel {
	result := make([]ConnectionModel, 0, len(pairs))
	for _, pair := range pairs {
		if len(pair) == 2 {
			result = append(result, ConnectionModel{
				Type:  types.StringValue(pair[0]),
				Value: types.StringValue(pair[1]),
			})
		}
	}
	return result
}

func stringPtrValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
