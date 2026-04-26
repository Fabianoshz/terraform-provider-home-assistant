package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEntityResource_basic(t *testing.T) {
	deviceID := "device-1"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		EntityRegistry: []testutil.MockEntityRegistry{
			{EntityID: "sensor.temperature", DeviceID: &deviceID, Platform: "esphome"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_entity" "temp" {
  entity_id = "sensor.temperature"
  name      = "Living Room Temperature"
  icon      = "mdi:thermometer"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_entity.temp", "id", "sensor.temperature"),
					resource.TestCheckResourceAttr("homeassistant_entity.temp", "name", "Living Room Temperature"),
					resource.TestCheckResourceAttr("homeassistant_entity.temp", "icon", "mdi:thermometer"),
					resource.TestCheckResourceAttr("homeassistant_entity.temp", "disabled", "false"),
				),
			},
		},
	})
}

func TestEntityResource_withArea(t *testing.T) {
	deviceID := "device-1"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Areas: []testutil.MockArea{
			{AreaID: "kitchen", Name: "Kitchen"},
		},
		EntityRegistry: []testutil.MockEntityRegistry{
			{EntityID: "sensor.temperature", DeviceID: &deviceID, Platform: "esphome"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_entity" "temp" {
  entity_id = "sensor.temperature"
  area_id   = "kitchen"
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_entity.temp", "area_id", "kitchen"),
			},
		},
	})
}

func TestEntityResource_disable(t *testing.T) {
	deviceID := "device-1"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		EntityRegistry: []testutil.MockEntityRegistry{
			{EntityID: "sensor.temperature", DeviceID: &deviceID, Platform: "esphome"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_entity" "temp" {
  entity_id = "sensor.temperature"
  disabled  = true
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_entity.temp", "disabled", "true"),
			},
		},
	})
}

func TestEntityResource_update(t *testing.T) {
	deviceID := "device-1"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		EntityRegistry: []testutil.MockEntityRegistry{
			{EntityID: "sensor.temperature", DeviceID: &deviceID, Platform: "esphome"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_entity" "temp" {
  entity_id = "sensor.temperature"
  name      = "Old Name"
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_entity.temp", "name", "Old Name"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_entity" "temp" {
  entity_id = "sensor.temperature"
  name      = "New Name"
  icon      = "mdi:thermometer"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_entity.temp", "name", "New Name"),
					resource.TestCheckResourceAttr("homeassistant_entity.temp", "icon", "mdi:thermometer"),
				),
			},
		},
	})
}
