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

var _ resource.Resource = &EntityResource{}

type EntityResource struct {
	client *client.Client
}

func NewEntityResource() resource.Resource {
	return &EntityResource{}
}

type EntityResourceModel struct {
	ID       types.String `tfsdk:"id"`
	DeviceID types.String `tfsdk:"device_id"`
	EntityID types.String `tfsdk:"entity_id"`
	Name     types.String `tfsdk:"name"`
	Icon     types.String `tfsdk:"icon"`
	AreaID   types.String `tfsdk:"area_id"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

func (r *EntityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entity"
}

func (r *EntityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages user-configurable properties of an existing Home Assistant entity.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the device this entity belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"entity_id": schema.StringAttribute{
				Required:    true,
				Description: "The Home Assistant entity ID (e.g. sensor.living_room_temperature).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "User-defined name override for the entity.",
			},
			"icon": schema.StringAttribute{
				Optional:    true,
				Description: "Custom MDI icon (e.g. \"mdi:thermometer\").",
			},
			"area_id": schema.StringAttribute{
				Optional:    true,
				Description: "Area to assign this entity to.",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the entity is enabled. When false, sets disabled_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *EntityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EntityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EntityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := r.client.GetEntityRegistryEntry(ctx, plan.EntityID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read entity", err.Error())
		return
	}
	if existing == nil {
		resp.Diagnostics.AddError("Entity not found", fmt.Sprintf("Entity %q does not exist in the registry.", plan.EntityID.ValueString()))
		return
	}
	if existing.DeviceID == nil || *existing.DeviceID != plan.DeviceID.ValueString() {
		resp.Diagnostics.AddError("Entity does not belong to device", fmt.Sprintf("Entity %q does not belong to device %q.", plan.EntityID.ValueString(), plan.DeviceID.ValueString()))
		return
	}

	entry, err := r.client.UpdateEntity(ctx, r.toUpdate(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure entity", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(entry))...)
}

func (r *EntityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EntityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, err := r.client.GetEntityRegistryEntry(ctx, state.EntityID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read entity", err.Error())
		return
	}
	if entry == nil || entry.DeviceID == nil || *entry.DeviceID != state.DeviceID.ValueString() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(entry))...)
}

func (r *EntityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EntityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, err := r.client.UpdateEntity(ctx, r.toUpdate(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update entity", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(entry))...)
}

func (r *EntityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EntityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clear user-managed fields; entities cannot be deleted via API.
	_, err := r.client.UpdateEntity(ctx, client.EntityUpdate{
		EntityID:      state.EntityID.ValueString(),
		SetDisabledBy: true,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset entity on destroy", err.Error())
	}
}

func (r *EntityResource) toUpdate(m EntityResourceModel) client.EntityUpdate {
	update := client.EntityUpdate{
		EntityID: m.EntityID.ValueString(),
		Name:     valueOrNil(m.Name),
		Icon:     valueOrNil(m.Icon),
		AreaID:   valueOrNil(m.AreaID),
	}
	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		update.SetDisabledBy = true
		if !m.Enabled.ValueBool() {
			update.DisabledBy = strPtr("user")
		}
	}
	return update
}

func (r *EntityResource) toModel(e *client.EntityRegistryEntry) EntityResourceModel {
	return EntityResourceModel{
		ID:       types.StringValue(e.EntityID),
		DeviceID: types.StringPointerValue(e.DeviceID),
		EntityID: types.StringValue(e.EntityID),
		Name:     stringPtrValue(e.Name),
		Icon:     stringPtrValue(e.Icon),
		AreaID:   stringPtrValue(e.AreaID),
		Enabled:  types.BoolValue(e.DisabledBy == nil),
	}
}
