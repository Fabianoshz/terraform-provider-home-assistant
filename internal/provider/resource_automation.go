package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &AutomationResource{}
	_ resource.ResourceWithValidateConfig = &AutomationResource{}
)

type AutomationResource struct {
	client *client.Client
}

func NewAutomationResource() resource.Resource {
	return &AutomationResource{}
}

func (r *AutomationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation"
}

func (r *AutomationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Home Assistant automation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"alias":       schema.StringAttribute{Required: true, Description: "Display name of the automation."},
			"description": schema.StringAttribute{Optional: true, Computed: true, Description: "Description of what the automation does."},
			"mode":        schema.StringAttribute{Optional: true, Computed: true, Description: "Automation mode: single, restart, queued, or parallel. Defaults to single."},
			"trigger":   schema.StringAttribute{Optional: true, Description: "JSON-encoded list of triggers. Required unless blueprint_path is set; conflicts with blueprint_path."},
			"condition": schema.StringAttribute{Optional: true, Computed: true, Description: "JSON-encoded list of conditions. Conflicts with blueprint_path."},
			"action":    schema.StringAttribute{Optional: true, Description: "JSON-encoded list of actions. Required unless blueprint_path is set; conflicts with blueprint_path."},
			"blueprint_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path of the blueprint to base this automation on (relative to the automation blueprints directory, e.g. \"my_blueprint.yaml\"). Mutually exclusive with trigger/condition/action.",
			},
			"blueprint_input": schema.StringAttribute{
				Optional:    true,
				Description: "JSON-encoded object mapping blueprint input names to values. Requires blueprint_path.",
			},
			"area_id":   schema.StringAttribute{Optional: true, Description: "Area to assign this automation to."},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the automation is enabled. When false, sets disabled_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"visible": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the automation is visible. When false, sets hidden_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AutomationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AutomationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AutomationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	usingBlueprint := !data.BlueprintPath.IsNull() && !data.BlueprintPath.IsUnknown()

	if usingBlueprint {
		if !data.Trigger.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Conflicting configuration", "trigger cannot be set when blueprint_path is used.")
		}
		if !data.Condition.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("condition"), "Conflicting configuration", "condition cannot be set when blueprint_path is used.")
		}
		if !data.Action.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("action"), "Conflicting configuration", "action cannot be set when blueprint_path is used.")
		}
		return
	}

	if !data.BlueprintInput.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("blueprint_input"), "Missing blueprint_path", "blueprint_input requires blueprint_path to be set.")
	}
	if data.Trigger.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("trigger"), "Missing required argument", "trigger is required unless blueprint_path is set.")
	}
	if data.Action.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("action"), "Missing required argument", "action is required unless blueprint_path is set.")
	}
}

func (r *AutomationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AutomationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	a, err := modelToAutomation(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid automation config", err.Error())
		return
	}

	created, err := r.client.CreateAutomation(ctx, *a)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create automation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, automationToModel(created))...)
}

func (r *AutomationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AutomationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	automation, err := r.client.GetAutomation(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read automation", err.Error())
		return
	}
	if automation == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, automationToModel(automation))...)
}

func (r *AutomationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AutomationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AutomationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	a, err := modelToAutomation(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid automation config", err.Error())
		return
	}
	a.ID = state.ID.ValueString()

	updated, err := r.client.UpdateAutomation(ctx, *a)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update automation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, automationToModel(updated))...)
}

func (r *AutomationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AutomationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAutomation(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete automation", err.Error())
	}
}

func automationToModel(a *client.Automation) AutomationModel {
	mode := a.Mode
	if mode == "" {
		mode = "single"
	}
	m := AutomationModel{
		ID:             types.StringValue(a.ID),
		Alias:          types.StringValue(a.Alias),
		Description:    types.StringValue(a.Description),
		Mode:           types.StringValue(mode),
		Trigger:        types.StringNull(),
		Condition:      types.StringNull(),
		Action:         types.StringNull(),
		BlueprintPath:  types.StringNull(),
		BlueprintInput: types.StringNull(),
		AreaID:         stringPtrValue(a.AreaID),
		Enabled:        boolPtrValue(a.Enabled),
		Visible:        boolPtrValue(a.Visible),
	}
	if a.UseBlueprint != nil {
		m.BlueprintPath = types.StringValue(a.UseBlueprint.Path)
		if len(a.UseBlueprint.Input) > 0 {
			m.BlueprintInput = types.StringValue(normalizeJSON(a.UseBlueprint.Input))
		}
	} else {
		m.Trigger = types.StringValue(rawToString(a.Trigger, "[]"))
		m.Condition = types.StringValue(rawToString(a.Condition, "[]"))
		m.Action = types.StringValue(rawToString(a.Action, "[]"))
	}
	return m
}

func modelToAutomation(m AutomationModel) (*client.Automation, error) {
	mode := m.Mode.ValueString()
	if mode == "" {
		mode = "single"
	}
	a := &client.Automation{
		Alias:       m.Alias.ValueString(),
		Description: m.Description.ValueString(),
		Mode:        mode,
		AreaID:      valueOrNil(m.AreaID),
		Enabled:     boolValueOrNil(m.Enabled),
		Visible:     boolValueOrNil(m.Visible),
	}

	if !m.BlueprintPath.IsNull() && !m.BlueprintPath.IsUnknown() {
		use := &client.UseBlueprint{Path: m.BlueprintPath.ValueString()}
		if !m.BlueprintInput.IsNull() && m.BlueprintInput.ValueString() != "" {
			input, err := parseRawJSON(m.BlueprintInput.ValueString(), "blueprint_input")
			if err != nil {
				return nil, err
			}
			use.Input = input
		}
		a.UseBlueprint = use
		return a, nil
	}

	trigger, err := parseRawJSON(m.Trigger.ValueString(), "trigger")
	if err != nil {
		return nil, err
	}
	condition, err := parseRawJSON(m.Condition.ValueString(), "condition")
	if err != nil {
		return nil, err
	}
	action, err := parseRawJSON(m.Action.ValueString(), "action")
	if err != nil {
		return nil, err
	}
	a.Trigger = trigger
	a.Condition = condition
	a.Action = action
	return a, nil
}

func normalizeJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(b)
}

func boolPtrValue(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

func boolValueOrNil(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

func rawToString(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 || string(raw) == "null" {
		return fallback
	}
	return string(raw)
}

func parseRawJSON(s, field string) (json.RawMessage, error) {
	if s == "" {
		return json.RawMessage("[]"), nil
	}
	if !json.Valid([]byte(s)) {
		return nil, fmt.Errorf("%s must be valid JSON", field)
	}
	return json.RawMessage(s), nil
}
