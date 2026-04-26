package provider

import (
	"context"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DeviceResource{}

type DeviceResource struct {
	client *client.Client
}

func NewDeviceResource() resource.Resource {
	return &DeviceResource{}
}

type DeviceResourceModel struct {
	ID          types.String `tfsdk:"id"`
	DeviceID    types.String `tfsdk:"device_id"`
	NameByUser  types.String `tfsdk:"name_by_user"`
	AreaID      types.String `tfsdk:"area_id"`
	Disabled    types.Bool   `tfsdk:"disabled"`
}

func (r *DeviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (r *DeviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages user-configurable properties of an existing Home Assistant device.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_id": schema.StringAttribute{
				Required:    true,
				Description: "The Home Assistant device ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name_by_user": schema.StringAttribute{
				Optional:    true,
				Description: "Custom name for the device, overriding the integration-provided name.",
			},
			"area_id": schema.StringAttribute{
				Optional:    true,
				Description: "Area to assign this device to.",
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the device is disabled. When true, sets disabled_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DeviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device, err := r.client.UpdateDevice(ctx, r.toUpdate(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure device", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(device, plan))...)
}

func (r *DeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device, err := r.client.GetDevice(ctx, state.DeviceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read device", err.Error())
		return
	}
	if device == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(device, state))...)
}

func (r *DeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device, err := r.client.UpdateDevice(ctx, r.toUpdate(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update device", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(device, plan))...)
}

func (r *DeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DeviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clear user-managed fields; the device itself cannot be deleted via API.
	_, err := r.client.UpdateDevice(ctx, client.DeviceUpdate{
		DeviceID:   state.DeviceID.ValueString(),
		NameByUser: nil,
		AreaID:     nil,
		DisabledBy: nil,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset device on destroy", err.Error())
	}
}

func (r *DeviceResource) toUpdate(m DeviceResourceModel) client.DeviceUpdate {
	update := client.DeviceUpdate{
		DeviceID:   m.DeviceID.ValueString(),
		NameByUser: valueOrNil(m.NameByUser),
		AreaID:     valueOrNil(m.AreaID),
	}
	if !m.Disabled.IsNull() && !m.Disabled.IsUnknown() {
		if m.Disabled.ValueBool() {
			update.DisabledBy = strPtr("user")
		}
	}
	return update
}

func (r *DeviceResource) toModel(d *client.Device, plan DeviceResourceModel) DeviceResourceModel {
	return DeviceResourceModel{
		ID:         types.StringValue(d.ID),
		DeviceID:   types.StringValue(d.ID),
		NameByUser: stringPtrValue(d.NameByUser),
		AreaID:     stringPtrValue(d.AreaID),
		Disabled:   types.BoolValue(d.DisabledBy != nil),
	}
}

func valueOrNil(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	v := s.ValueString()
	return &v
}

func strPtr(s string) *string { return &s }
