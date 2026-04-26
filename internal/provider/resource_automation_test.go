package provider_test

import (
	"encoding/json"
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
					resource.TestCheckResourceAttr("homeassistant_automation.motion_lights", "id", "automation_1"),
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
