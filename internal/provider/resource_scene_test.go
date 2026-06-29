package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestSceneResource_create(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_scene" "movie" {
  name = "Movie Time"
  icon = "mdi:movie"
  entities = jsonencode({
    "light.living_room" = { state = "on", brightness = 50 }
    "light.kitchen"     = "off"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("homeassistant_scene.movie", "id"),
					resource.TestCheckResourceAttr("homeassistant_scene.movie", "name", "Movie Time"),
					resource.TestCheckResourceAttr("homeassistant_scene.movie", "icon", "mdi:movie"),
				),
			},
		},
	})
}

func TestSceneResource_update(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_scene" "movie" {
  name     = "Old Name"
  entities = jsonencode({ "light.living_room" = "on" })
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_scene.movie", "name", "Old Name"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_scene" "movie" {
  name     = "New Name"
  entities = jsonencode({ "light.living_room" = "off" })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_scene.movie", "name", "New Name"),
					resource.TestCheckResourceAttr("homeassistant_scene.movie", "entities", `{"light.living_room":"off"}`),
				),
			},
		},
	})
}
