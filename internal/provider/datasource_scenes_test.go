package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScenesDataSource(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		EntityStates: []testutil.MockEntityState{
			{
				EntityID: "scene.movie_time",
				State:    "2024-01-01T00:00:00Z",
				Attributes: map[string]any{
					"id":            "1700000000000",
					"friendly_name": "Movie Time",
					"entity_id":     []any{"light.living_room", "light.kitchen"},
				},
			},
			{EntityID: "light.living_room", State: "on"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_scenes" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_scenes.all", "scenes.#", "1"),
					resource.TestCheckResourceAttr("data.homeassistant_scenes.all", "scenes.0.id", "1700000000000"),
					resource.TestCheckResourceAttr("data.homeassistant_scenes.all", "scenes.0.entity_id", "scene.movie_time"),
					resource.TestCheckResourceAttr("data.homeassistant_scenes.all", "scenes.0.name", "Movie Time"),
					resource.TestCheckResourceAttr("data.homeassistant_scenes.all", "scenes.0.entities.#", "2"),
					resource.TestCheckResourceAttr("data.homeassistant_scenes.all", "scenes.0.entities.0", "light.living_room"),
				),
			},
		},
	})
}
