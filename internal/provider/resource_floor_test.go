package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestFloorResource_create(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_floor" "ground" {
  name = "Ground Floor"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_floor.ground", "floor_id", "ground_floor"),
					resource.TestCheckResourceAttr("homeassistant_floor.ground", "name", "Ground Floor"),
				),
			},
		},
	})
}

func TestFloorResource_withLevelAndIcon(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_floor" "first" {
  name  = "First Floor"
  level = 1
  icon  = "mdi:home-floor-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_floor.first", "floor_id", "first_floor"),
					resource.TestCheckResourceAttr("homeassistant_floor.first", "name", "First Floor"),
					resource.TestCheckResourceAttr("homeassistant_floor.first", "level", "1"),
					resource.TestCheckResourceAttr("homeassistant_floor.first", "icon", "mdi:home-floor-1"),
				),
			},
		},
	})
}

func TestFloorResource_update(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_floor" "f" {
  name  = "Old Name"
  level = 0
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_floor.f", "name", "Old Name"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_floor" "f" {
  name  = "New Name"
  level = 0
  icon  = "mdi:home"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_floor.f", "name", "New Name"),
					resource.TestCheckResourceAttr("homeassistant_floor.f", "icon", "mdi:home"),
				),
			},
		},
	})
}

func TestFloorResource_usedInArea(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_floor" "ground" {
  name  = "Ground Floor"
  level = 0
}

resource "homeassistant_area" "living_room" {
  name     = "Living Room"
  floor_id = homeassistant_floor.ground.floor_id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_floor.ground", "floor_id", "ground_floor"),
					resource.TestCheckResourceAttr("homeassistant_area.living_room", "area_id", "living_room"),
					resource.TestCheckResourceAttr("homeassistant_area.living_room", "floor_id", "ground_floor"),
				),
			},
		},
	})
}
