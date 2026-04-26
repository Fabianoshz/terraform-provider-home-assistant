package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAreaResource_create(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_area" "kitchen" {
  name = "Kitchen"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_area.kitchen", "area_id", "kitchen"),
					resource.TestCheckResourceAttr("homeassistant_area.kitchen", "name", "Kitchen"),
				),
			},
		},
	})
}

func TestAreaResource_withIcon(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_area" "living_room" {
  name = "Living Room"
  icon = "mdi:sofa"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_area.living_room", "area_id", "living_room"),
					resource.TestCheckResourceAttr("homeassistant_area.living_room", "icon", "mdi:sofa"),
				),
			},
		},
	})
}

func TestAreaResource_update(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_area" "room" {
  name = "Old Name"
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_area.room", "name", "Old Name"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_area" "room" {
  name = "New Name"
  icon = "mdi:home"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_area.room", "name", "New Name"),
					resource.TestCheckResourceAttr("homeassistant_area.room", "icon", "mdi:home"),
				),
			},
		},
	})
}

func TestAreasDataSource(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		Areas: []testutil.MockArea{
			{AreaID: "kitchen", Name: "Kitchen", Icon: testutil.Ptr("mdi:countertop")},
			{AreaID: "living_room", Name: "Living Room"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_areas" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_areas.all", "areas.#", "2"),
					resource.TestCheckResourceAttr("data.homeassistant_areas.all", "areas.0.area_id", "kitchen"),
					resource.TestCheckResourceAttr("data.homeassistant_areas.all", "areas.0.name", "Kitchen"),
					resource.TestCheckResourceAttr("data.homeassistant_areas.all", "areas.0.icon", "mdi:countertop"),
					resource.TestCheckResourceAttr("data.homeassistant_areas.all", "areas.1.area_id", "living_room"),
				),
			},
		},
	})
}

func TestAreaResource_usedInDevice(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token:   "test-token",
		Devices: []testutil.MockDevice{{ID: "device-1", Name: "ESP32"}},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_area" "kitchen" {
  name = "Kitchen"
}

resource "homeassistant_device" "esp" {
  device_id = "device-1"
  area_id   = homeassistant_area.kitchen.area_id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_area.kitchen", "area_id", "kitchen"),
					resource.TestCheckResourceAttr("homeassistant_device.esp", "area_id", "kitchen"),
				),
			},
		},
	})
}
