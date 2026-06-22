package provider_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAutomationResource_create(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_automation" "motion_lights" {
  alias   = "Turn on lights on motion"
  trigger = jsonencode([{platform = "state", entity_id = "binary_sensor.motion", to = "on"}])
  action  = jsonencode([{service = "light.turn_on", target = {entity_id = "light.living_room"}}])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("homeassistant_automation.motion_lights", "id"),
					resource.TestCheckResourceAttr("homeassistant_automation.motion_lights", "alias", "Turn on lights on motion"),
					resource.TestCheckResourceAttr("homeassistant_automation.motion_lights", "mode", "single"),
				),
			},
		},
	})
}

func TestAutomationResource_withDescription(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_automation" "motion_lights" {
  alias       = "Motion Lights"
  description = "Turns on lights when motion is detected"
  mode        = "restart"
  trigger     = jsonencode([{platform = "state", entity_id = "binary_sensor.motion", to = "on"}])
  action      = jsonencode([{service = "light.turn_on", target = {entity_id = "light.living_room"}}])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_automation.motion_lights", "description", "Turns on lights when motion is detected"),
					resource.TestCheckResourceAttr("homeassistant_automation.motion_lights", "mode", "restart"),
				),
			},
		},
	})
}

func TestAutomationResource_update(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_automation" "motion_lights" {
  alias   = "Old Alias"
  trigger = jsonencode([{platform = "state", entity_id = "binary_sensor.motion", to = "on"}])
  action  = jsonencode([{service = "light.turn_on", target = {entity_id = "light.living_room"}}])
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_automation.motion_lights", "alias", "Old Alias"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_automation" "motion_lights" {
  alias   = "New Alias"
  trigger = jsonencode([{platform = "state", entity_id = "binary_sensor.motion", to = "on"}])
  action  = jsonencode([{service = "light.turn_on", target = {entity_id = "light.living_room"}}])
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_automation.motion_lights", "alias", "New Alias"),
			},
		},
	})
}

func TestAutomationResource_useBlueprint(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_automation" "from_blueprint" {
  alias           = "Motion Light From Blueprint"
  blueprint_path  = "motion_light.yaml"
  blueprint_input = jsonencode({
    motion_entity = "binary_sensor.hallway"
    light_target  = { entity_id = "light.hallway" }
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("homeassistant_automation.from_blueprint", "id"),
					resource.TestCheckResourceAttr("homeassistant_automation.from_blueprint", "alias", "Motion Light From Blueprint"),
					resource.TestCheckResourceAttr("homeassistant_automation.from_blueprint", "blueprint_path", "motion_light.yaml"),
					resource.TestCheckResourceAttr("homeassistant_automation.from_blueprint", "blueprint_input", `{"light_target":{"entity_id":"light.hallway"},"motion_entity":"binary_sensor.hallway"}`),
					resource.TestCheckNoResourceAttr("homeassistant_automation.from_blueprint", "trigger"),
					resource.TestCheckNoResourceAttr("homeassistant_automation.from_blueprint", "action"),
				),
			},
		},
	})
}

func TestAutomationResource_blueprintConflictsWithTrigger(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_automation" "bad" {
  alias          = "Bad"
  blueprint_path = "motion_light.yaml"
  trigger        = jsonencode([{platform = "state", entity_id = "binary_sensor.motion"}])
}
`,
				ExpectError: regexp.MustCompile(`Conflicting configuration`),
			},
		},
	})
}

func TestAutomationResource_requiresTriggerWithoutBlueprint(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_automation" "bad" {
  alias  = "Bad"
  action = jsonencode([{service = "light.turn_on"}])
}
`,
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
		},
	})
}

func TestAutomationsDataSource(t *testing.T) {
	trigger, _ := json.Marshal([]map[string]any{{"platform": "state", "entity_id": "binary_sensor.motion"}})
	action, _ := json.Marshal([]map[string]any{{"service": "light.turn_on"}})

	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Automations: []testutil.MockAutomation{
			{ID: "abc123", Alias: "Motion Lights", Mode: "single", Trigger: trigger, Action: action},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_automations" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_automations.all", "automations.#", "1"),
					resource.TestCheckResourceAttr("data.homeassistant_automations.all", "automations.0.id", "abc123"),
					resource.TestCheckResourceAttr("data.homeassistant_automations.all", "automations.0.alias", "Motion Lights"),
					resource.TestCheckResourceAttr("data.homeassistant_automations.all", "automations.0.mode", "single"),
				),
			},
		},
	})
}
