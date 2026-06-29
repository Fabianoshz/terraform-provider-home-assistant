package provider

import (
	"context"
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
	_ resource.Resource                   = &ScriptResource{}
	_ resource.ResourceWithValidateConfig = &ScriptResource{}
)

type ScriptResource struct {
	client *client.Client
}

type ScriptModel struct {
	ID             types.String `tfsdk:"id"`
	Alias          types.String `tfsdk:"alias"`
	Description    types.String `tfsdk:"description"`
	Icon           types.String `tfsdk:"icon"`
	Mode           types.String `tfsdk:"mode"`
	Sequence       types.String `tfsdk:"sequence"`
	Fields         types.String `tfsdk:"fields"`
	BlueprintPath  types.String `tfsdk:"blueprint_path"`
	BlueprintInput types.String `tfsdk:"blueprint_input"`
	AreaID         types.String `tfsdk:"area_id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Visible        types.Bool   `tfsdk:"visible"`
}

func NewScriptResource() resource.Resource {
	return &ScriptResource{}
}

func (r *ScriptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script"
}

func (r *ScriptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Home Assistant script.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Object ID of the script; the entity is exposed as script.<id>. If omitted, a value is generated. Use a slug (lowercase letters, digits and underscores) for a friendly entity ID. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alias":       schema.StringAttribute{Required: true, Description: "Display name of the script."},
			"description": schema.StringAttribute{Optional: true, Computed: true, Description: "Description of what the script does."},
			"icon":        schema.StringAttribute{Optional: true, Computed: true, Description: "Icon for the script (e.g. \"mdi:script-text\")."},
			"mode":        schema.StringAttribute{Optional: true, Computed: true, Description: "Script mode: single, restart, queued, or parallel. Defaults to single."},
			"sequence":    schema.StringAttribute{Optional: true, Description: "JSON-encoded list of actions the script runs. Required unless blueprint_path is set; conflicts with blueprint_path."},
			"fields":      schema.StringAttribute{Optional: true, Description: "JSON-encoded object describing the script's input fields. Conflicts with blueprint_path."},
			"blueprint_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path of the blueprint to base this script on (relative to the script blueprints directory, e.g. \"confirmable_notification.yaml\"). Mutually exclusive with sequence/fields.",
			},
			"blueprint_input": schema.StringAttribute{
				Optional:    true,
				Description: "JSON-encoded object mapping blueprint input names to values. Requires blueprint_path.",
			},
			"area_id": schema.StringAttribute{Optional: true, Description: "Area to assign this script to."},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the script is enabled. When false, sets disabled_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"visible": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the script is visible. When false, sets hidden_by to \"user\".",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ScriptResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScriptResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ScriptModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When wrapped in a module these attributes can be Unknown at validate time
	// (variables aren't resolved yet); defer cross-field rules to plan/apply.
	if data.BlueprintPath.IsUnknown() || data.BlueprintInput.IsUnknown() ||
		data.Sequence.IsUnknown() || data.Fields.IsUnknown() {
		return
	}

	usingBlueprint := !data.BlueprintPath.IsNull()

	if usingBlueprint {
		if !data.Sequence.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("sequence"), "Conflicting configuration", "sequence cannot be set when blueprint_path is used.")
		}
		if !data.Fields.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("fields"), "Conflicting configuration", "fields cannot be set when blueprint_path is used.")
		}
		return
	}

	if !data.BlueprintInput.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("blueprint_input"), "Missing blueprint_path", "blueprint_input requires blueprint_path to be set.")
	}
	if data.Sequence.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("sequence"), "Missing required argument", "sequence is required unless blueprint_path is set.")
	}
}

func (r *ScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := modelToScript(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid script config", err.Error())
		return
	}

	created, err := r.client.CreateScript(ctx, *s)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create script", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scriptToModel(created))...)
}

func (r *ScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	script, err := r.client.GetScript(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}
	if script == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scriptToModel(script))...)
}

func (r *ScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := modelToScript(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid script config", err.Error())
		return
	}
	s.ObjectID = state.ID.ValueString()

	updated, err := r.client.UpdateScript(ctx, *s)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update script", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scriptToModel(updated))...)
}

func (r *ScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteScript(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete script", err.Error())
	}
}

func scriptToModel(s *client.Script) ScriptModel {
	mode := s.Mode
	if mode == "" {
		mode = "single"
	}
	m := ScriptModel{
		ID:             types.StringValue(s.ObjectID),
		Alias:          types.StringValue(s.Alias),
		Description:    types.StringValue(s.Description),
		Icon:           types.StringValue(s.Icon),
		Mode:           types.StringValue(mode),
		Sequence:       types.StringNull(),
		Fields:         types.StringNull(),
		BlueprintPath:  types.StringNull(),
		BlueprintInput: types.StringNull(),
		AreaID:         stringPtrValue(s.AreaID),
		Enabled:        boolPtrValue(s.Enabled),
		Visible:        boolPtrValue(s.Visible),
	}
	if s.UseBlueprint != nil {
		m.BlueprintPath = types.StringValue(s.UseBlueprint.Path)
		if len(s.UseBlueprint.Input) > 0 {
			m.BlueprintInput = types.StringValue(normalizeJSON(s.UseBlueprint.Input))
		}
	} else {
		m.Sequence = types.StringValue(rawToString(s.Sequence, "[]"))
		if len(s.Fields) > 0 {
			m.Fields = types.StringValue(normalizeJSON(s.Fields))
		}
	}
	return m
}

func modelToScript(m ScriptModel) (*client.Script, error) {
	mode := m.Mode.ValueString()
	if mode == "" {
		mode = "single"
	}
	s := &client.Script{
		Alias:       m.Alias.ValueString(),
		Description: m.Description.ValueString(),
		Icon:        m.Icon.ValueString(),
		Mode:        mode,
		AreaID:      valueOrNil(m.AreaID),
		Enabled:     boolValueOrNil(m.Enabled),
		Visible:     boolValueOrNil(m.Visible),
	}
	// A user-supplied id is used as the object_id; when omitted it is Unknown
	// here and the client generates one on create.
	if !m.ID.IsNull() && !m.ID.IsUnknown() {
		s.ObjectID = m.ID.ValueString()
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
		s.UseBlueprint = use
		return s, nil
	}

	sequence, err := parseRawJSON(m.Sequence.ValueString(), "sequence")
	if err != nil {
		return nil, err
	}
	s.Sequence = sequence
	if !m.Fields.IsNull() && !m.Fields.IsUnknown() && m.Fields.ValueString() != "" {
		fields, err := parseRawJSON(m.Fields.ValueString(), "fields")
		if err != nil {
			return nil, err
		}
		s.Fields = fields
	}
	return s, nil
}
