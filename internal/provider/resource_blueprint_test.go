package provider_test

import (
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestBlueprintResource_create(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_blueprint" "motion" {
  domain = "automation"
  path   = "my_dir/motion_light.yaml"
  blueprint = <<-EOT
    blueprint:
      name: Motion Light
      domain: automation
  EOT
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_blueprint.motion", "id", "automation/my_dir/motion_light.yaml"),
					resource.TestCheckResourceAttr("homeassistant_blueprint.motion", "domain", "automation"),
					resource.TestCheckResourceAttr("homeassistant_blueprint.motion", "path", "my_dir/motion_light.yaml"),
				),
			},
		},
	})
}

func TestBlueprintResource_withSourceURL(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_blueprint" "imported" {
  domain     = "automation"
  path       = "imported.yaml"
  source_url = "https://example.com/blueprint.yaml"
  blueprint  = "blueprint:\n  name: Imported\n  domain: automation\n"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_blueprint.imported", "source_url", "https://example.com/blueprint.yaml"),
					resource.TestCheckResourceAttr("homeassistant_blueprint.imported", "id", "automation/imported.yaml"),
				),
			},
		},
	})
}

func TestBlueprintResource_update(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_blueprint" "bp" {
  domain    = "automation"
  path      = "bp.yaml"
  blueprint = "blueprint:\n  name: Old\n  domain: automation\n"
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_blueprint.bp", "blueprint", "blueprint:\n  name: Old\n  domain: automation\n"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_blueprint" "bp" {
  domain    = "automation"
  path      = "bp.yaml"
  blueprint = "blueprint:\n  name: New\n  domain: automation\n"
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_blueprint.bp", "blueprint", "blueprint:\n  name: New\n  domain: automation\n"),
			},
		},
	})
}
