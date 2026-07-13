package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
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
	ID        types.String `tfsdk:"id"`
	DeviceID  types.String `tfsdk:"device_id"`
	EntityID  types.String `tfsdk:"entity_id"`
	Name      types.String `tfsdk:"name"`
	Icon      types.String `tfsdk:"icon"`
	AreaID    types.String `tfsdk:"area_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	Visible   types.Bool   `tfsdk:"visible"`
	Aliases   types.Set    `tfsdk:"aliases"`
	Labels    types.Set    `tfsdk:"labels"`
	ExposedTo types.Set    `tfsdk:"exposed_to"`
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
				Computed:    true,
				Description: "User-defined name override for the entity. Set to \"\" to clear.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"icon": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Custom MDI icon (e.g. \"mdi:thermometer\"). Set to \"\" to clear.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"area_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Area to assign this entity to. Set to \"\" to clear.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the entity is enabled. When false, sets disabled_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"visible": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the entity is visible. When false, sets hidden_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"aliases": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Voice-assistant aliases (alternative names) for the entity.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Label IDs assigned to the entity.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"exposed_to": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Assistants the entity is exposed to (e.g. \"conversation\", \"cloud.alexa\", \"cloud.google_assistant\"). When set, manages the complete exposure set for the entity.",
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

	update := r.toUpdate(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	entry, err := r.client.UpdateEntity(ctx, update)
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure entity", err.Error())
		return
	}

	r.applyExposure(ctx, plan.EntityID.ValueString(), plan.ExposedTo, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	model := r.toModel(ctx, entry, &resp.Diagnostics)
	model.ExposedTo = plan.ExposedTo
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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

	model := r.toModel(ctx, entry, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Exposure lives in a separate registry; only track it when the field is managed.
	if !state.ExposedTo.IsNull() {
		current, err := r.client.GetExposedAssistants(ctx, state.EntityID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read entity exposure", err.Error())
			return
		}
		set, d := types.SetValueFrom(ctx, types.StringType, current)
		resp.Diagnostics.Append(d...)
		model.ExposedTo = set
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *EntityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EntityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := r.toUpdate(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	entry, err := r.client.UpdateEntity(ctx, update)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update entity", err.Error())
		return
	}

	r.applyExposure(ctx, plan.EntityID.ValueString(), plan.ExposedTo, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	model := r.toModel(ctx, entry, &resp.Diagnostics)
	model.ExposedTo = plan.ExposedTo
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
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
		SetName:       true,
		SetIcon:       true,
		SetAreaID:     true,
		SetDisabledBy: true,
		SetHiddenBy:   true,
		SetAliases:    true,
		SetLabels:     true,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset entity on destroy", err.Error())
	}

	if !state.ExposedTo.IsNull() {
		if current, err := r.client.GetExposedAssistants(ctx, state.EntityID.ValueString()); err == nil && len(current) > 0 {
			_ = r.client.SetEntityExposure(ctx, state.EntityID.ValueString(), current, false)
		}
	}
}

func (r *EntityResource) toUpdate(ctx context.Context, m EntityResourceModel, diags *diag.Diagnostics) client.EntityUpdate {
	update := client.EntityUpdate{EntityID: m.EntityID.ValueString()}
	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		update.SetName = true
		if m.Name.ValueString() != "" {
			update.Name = strPtr(m.Name.ValueString())
		}
	}
	if !m.Icon.IsNull() && !m.Icon.IsUnknown() {
		update.SetIcon = true
		if m.Icon.ValueString() != "" {
			update.Icon = strPtr(m.Icon.ValueString())
		}
	}
	if !m.AreaID.IsNull() && !m.AreaID.IsUnknown() {
		update.SetAreaID = true
		if m.AreaID.ValueString() != "" {
			update.AreaID = strPtr(m.AreaID.ValueString())
		}
	}
	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		update.SetDisabledBy = true
		if !m.Enabled.ValueBool() {
			update.DisabledBy = strPtr("user")
		}
	}
	if !m.Visible.IsNull() && !m.Visible.IsUnknown() {
		update.SetHiddenBy = true
		if !m.Visible.ValueBool() {
			update.HiddenBy = strPtr("user")
		}
	}
	if !m.Aliases.IsNull() && !m.Aliases.IsUnknown() {
		update.SetAliases = true
		diags.Append(m.Aliases.ElementsAs(ctx, &update.Aliases, false)...)
	}
	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		update.SetLabels = true
		diags.Append(m.Labels.ElementsAs(ctx, &update.Labels, false)...)
	}
	return update
}

// applyExposure makes the entity's exposure match the desired assistant set exactly.
func (r *EntityResource) applyExposure(ctx context.Context, entityID string, desired types.Set, diags *diag.Diagnostics) {
	if desired.IsNull() || desired.IsUnknown() {
		return
	}
	var want []string
	diags.Append(desired.ElementsAs(ctx, &want, false)...)
	if diags.HasError() {
		return
	}
	current, err := r.client.GetExposedAssistants(ctx, entityID)
	if err != nil {
		diags.AddError("Failed to read entity exposure", err.Error())
		return
	}
	wantSet := map[string]bool{}
	for _, a := range want {
		wantSet[a] = true
	}
	curSet := map[string]bool{}
	for _, a := range current {
		curSet[a] = true
	}
	var toExpose, toUnexpose []string
	for _, a := range want {
		if !curSet[a] {
			toExpose = append(toExpose, a)
		}
	}
	for _, a := range current {
		if !wantSet[a] {
			toUnexpose = append(toUnexpose, a)
		}
	}
	if len(toExpose) > 0 {
		if err := r.client.SetEntityExposure(ctx, entityID, toExpose, true); err != nil {
			diags.AddError("Failed to expose entity", err.Error())
			return
		}
	}
	if len(toUnexpose) > 0 {
		if err := r.client.SetEntityExposure(ctx, entityID, toUnexpose, false); err != nil {
			diags.AddError("Failed to unexpose entity", err.Error())
			return
		}
	}
}

func (r *EntityResource) toModel(ctx context.Context, e *client.EntityRegistryEntry, diags *diag.Diagnostics) EntityResourceModel {
	aliases, d := types.SetValueFrom(ctx, types.StringType, e.Aliases)
	diags.Append(d...)
	labels, d2 := types.SetValueFrom(ctx, types.StringType, e.Labels)
	diags.Append(d2...)
	return EntityResourceModel{
		ID:        types.StringValue(e.EntityID),
		DeviceID:  types.StringPointerValue(e.DeviceID),
		EntityID:  types.StringValue(e.EntityID),
		Name:      stringPtrValue(e.Name),
		Icon:      stringPtrValue(e.Icon),
		AreaID:    stringPtrValue(e.AreaID),
		Enabled:   types.BoolValue(e.DisabledBy == nil),
		Visible:   types.BoolValue(e.HiddenBy == nil),
		Aliases:   aliases,
		Labels:    labels,
		ExposedTo: types.SetNull(types.StringType),
	}
}
