package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDeviceResource_basic(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{ID: "device-1", Name: "ESP32 Board"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_device" "esp" {
  device_id    = "device-1"
  name_by_user = "Living Room Sensor"
  area_id      = "living_room"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_device.esp", "id", "device-1"),
					resource.TestCheckResourceAttr("homeassistant_device.esp", "name_by_user", "Living Room Sensor"),
					resource.TestCheckResourceAttr("homeassistant_device.esp", "area_id", "living_room"),
					resource.TestCheckResourceAttr("homeassistant_device.esp", "disabled", "false"),
				),
			},
		},
	})
}

func TestDeviceResource_update(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{ID: "device-1", Name: "ESP32 Board"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_device" "esp" {
  device_id    = "device-1"
  name_by_user = "Old Name"
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_device.esp", "name_by_user", "Old Name"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_device" "esp" {
  device_id    = "device-1"
  name_by_user = "New Name"
  area_id      = "kitchen"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_device.esp", "name_by_user", "New Name"),
					resource.TestCheckResourceAttr("homeassistant_device.esp", "area_id", "kitchen"),
				),
			},
		},
	})
}

func TestDeviceResource_disable(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{ID: "device-1", Name: "ESP32 Board"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_device" "esp" {
  device_id = "device-1"
  disabled  = true
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_device.esp", "disabled", "true"),
			},
		},
	})
}
