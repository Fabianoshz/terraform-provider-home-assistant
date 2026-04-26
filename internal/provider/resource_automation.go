package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AutomationResource{}

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
			"trigger":     schema.StringAttribute{Required: true, Description: "JSON-encoded list of triggers."},
			"condition":   schema.StringAttribute{Optional: true, Computed: true, Description: "JSON-encoded list of conditions."},
			"action":      schema.StringAttribute{Required: true, Description: "JSON-encoded list of actions."},
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
	return AutomationModel{
		ID:          types.StringValue(a.ID),
		Alias:       types.StringValue(a.Alias),
		Description: types.StringValue(a.Description),
		Mode:        types.StringValue(mode),
		Trigger:     types.StringValue(rawToString(a.Trigger, "[]")),
		Condition:   types.StringValue(rawToString(a.Condition, "[]")),
		Action:      types.StringValue(rawToString(a.Action, "[]")),
	}
}

func modelToAutomation(m AutomationModel) (*client.Automation, error) {
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
	mode := m.Mode.ValueString()
	if mode == "" {
		mode = "single"
	}
	return &client.Automation{
		Alias:       m.Alias.ValueString(),
		Description: m.Description.ValueString(),
		Mode:        mode,
		Trigger:     trigger,
		Condition:   condition,
		Action:      action,
	}, nil
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
