package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScriptsDataSource(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{
		Token: "test-token",
		EntityStates: []testutil.MockEntityState{
			{
				EntityID: "script.notify_everyone",
				State:    "off",
				Attributes: map[string]any{
					"friendly_name": "Notify Everyone",
					"mode":          "single",
				},
			},
			{EntityID: "automation.something", State: "on"},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
data "homeassistant_scripts" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.homeassistant_scripts.all", "scripts.#", "1"),
					resource.TestCheckResourceAttr("data.homeassistant_scripts.all", "scripts.0.id", "notify_everyone"),
					resource.TestCheckResourceAttr("data.homeassistant_scripts.all", "scripts.0.entity_id", "script.notify_everyone"),
					resource.TestCheckResourceAttr("data.homeassistant_scripts.all", "scripts.0.name", "Notify Everyone"),
					resource.TestCheckResourceAttr("data.homeassistant_scripts.all", "scripts.0.mode", "single"),
					resource.TestCheckResourceAttr("data.homeassistant_scripts.all", "scripts.0.state", "off"),
				),
			},
		},
	})
}
