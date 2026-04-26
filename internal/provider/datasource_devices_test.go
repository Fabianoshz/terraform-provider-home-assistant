package provider_test

import (
	"fmt"
	"testing"

	"github.com/Fabianoshz/terraform-provider-home-assistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDevicesDataSource_allDevices(t *testing.T) {
	deviceID := "device-1"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{
				ID:           deviceID,
				Name:         "Living Room Light",
				Manufacturer: testutil.Ptr("Philips"),
				Model:        testutil.Ptr("Hue Bulb"),
				Connections:  [][]string{{"mac", "aa:bb:cc:dd:ee:ff"}},
				Identifiers:  [][]string{{"hue", "hue-light-1"}},
			},
			{
				ID:          "device-2",
				Name:        "Temp Sensor",
				Identifiers: [][]string{{"zha", "00:12:4b:00:1a:2b:3c:4d"}},
			},
		},
		EntityRegistry: []testutil.MockEntityRegistry{
			{EntityID: "light.living_room", DeviceID: &deviceID, OriginalName: testutil.Ptr("Living Room"), Platform: "hue"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_devices" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.#", "2"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.id", "device-1"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.name", "Living Room Light"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.manufacturer", "Philips"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.connections.0.type", "mac"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.connections.0.value", "aa:bb:cc:dd:ee:ff"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.identifiers.0.type", "hue"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.entities.#", "1"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.entities.0.entity_id", "light.living_room"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.entities.0.original_name", "Living Room"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.entities.0.platform", "hue"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.1.id", "device-2"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.1.entities.#", "0"),
				),
			},
		},
	})
}

func TestDevicesDataSource_filterByConnection(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{
				ID:          "device-1",
				Name:        "Switch A",
				Connections: [][]string{{"mac", "aa:bb:cc:dd:ee:ff"}},
			},
			{
				ID:          "device-2",
				Name:        "Switch B",
				Connections: [][]string{{"mac", "11:22:33:44:55:66"}},
			},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_devices" "filtered" {
  filter {
    source = "connections"
    type   = "mac"
    value  = "aa:bb:cc:dd:ee:ff"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_devices.filtered", "devices.#", "1"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.filtered", "devices.0.id", "device-1"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.filtered", "devices.0.name", "Switch A"),
				),
			},
		},
	})
}

func TestDevicesDataSource_filterByIdentifier(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{
				ID:          "device-1",
				Name:        "Zigbee Sensor",
				Identifiers: [][]string{{"zha", "00:12:4b:00:1a:2b:3c:4d"}},
			},
			{
				ID:          "device-2",
				Name:        "Hue Light",
				Identifiers: [][]string{{"hue", "hue-light-42"}},
			},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_devices" "filtered" {
  filter {
    source = "identifiers"
    type   = "zha"
    value  = "00:12:4b:00:1a:2b:3c:4d"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_devices.filtered", "devices.#", "1"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.filtered", "devices.0.name", "Zigbee Sensor"),
				),
			},
		},
	})
}

func TestDevicesDataSource_filterNoMatch(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{ID: "device-1", Name: "Some Device", Connections: [][]string{{"mac", "aa:bb:cc:dd:ee:ff"}}},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_devices" "filtered" {
  filter {
    source = "connections"
    type   = "mac"
    value  = "00:00:00:00:00:00"
  }
}
`,
				Check: resource.TestCheckResourceAttr("data.homeassistant_devices.filtered", "devices.#", "0"),
			},
		},
	})
}

func TestDevicesDataSource_hideDisabledEntities(t *testing.T) {
	deviceID := "device-1"
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Devices: []testutil.MockDevice{
			{ID: deviceID, Name: "ESP32 Board"},
		},
		EntityRegistry: []testutil.MockEntityRegistry{
			{EntityID: "sensor.temperature", DeviceID: &deviceID, Platform: "esphome"},
			{EntityID: "sensor.humidity", DeviceID: &deviceID, Platform: "esphome", DisabledBy: testutil.Ptr("user")},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_devices" "all" {
  hide_disabled_entities = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.entities.#", "1"),
					resource.TestCheckResourceAttr("data.homeassistant_devices.all", "devices.0.entities.0.entity_id", "sensor.temperature"),
				),
			},
		},
	})
}

func providerConfig(url, token string) string {
	return fmt.Sprintf(`
provider "homeassistant" {
  url   = %q
  token = %q
}
`, url, token)
}
