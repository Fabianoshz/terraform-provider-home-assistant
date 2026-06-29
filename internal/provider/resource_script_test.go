package provider_test

import (
	"regexp"
	"testing"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/testutil"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestScriptResource_create(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "notify" {
  alias    = "Notify Everyone"
  icon     = "mdi:bell"
  sequence = jsonencode([{ service = "notify.notify", data = { message = "Hello" } }])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("homeassistant_script.notify", "id"),
					resource.TestCheckResourceAttr("homeassistant_script.notify", "alias", "Notify Everyone"),
					resource.TestCheckResourceAttr("homeassistant_script.notify", "icon", "mdi:bell"),
					resource.TestCheckResourceAttr("homeassistant_script.notify", "mode", "single"),
				),
			},
		},
	})
}

func TestScriptResource_withFields(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "notify" {
  alias    = "Notify"
  mode     = "queued"
  sequence = jsonencode([{ service = "notify.notify", data = { message = "{{ message }}" } }])
  fields   = jsonencode({ message = { name = "Message", required = true, selector = { text = {} } } })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("homeassistant_script.notify", "mode", "queued"),
					resource.TestCheckResourceAttr("homeassistant_script.notify", "fields", `{"message":{"name":"Message","required":true,"selector":{"text":{}}}}`),
				),
			},
		},
	})
}

func TestScriptResource_explicitID(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "notify" {
  id       = "notify_everyone"
  alias    = "Notify Everyone"
  sequence = jsonencode([{ action = "notify.notify" }])
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_script.notify", "id", "notify_everyone"),
			},
		},
	})
}

func TestScriptResource_changingIDForcesReplace(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "notify" {
  id       = "first_id"
  alias    = "Notify"
  sequence = jsonencode([{ action = "notify.notify" }])
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_script.notify", "id", "first_id"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "notify" {
  id       = "second_id"
  alias    = "Notify"
  sequence = jsonencode([{ action = "notify.notify" }])
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_script.notify", "id", "second_id"),
			},
		},
	})
}

func TestScriptResource_useBlueprint(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "from_blueprint" {
  alias           = "Confirmable Notification"
  blueprint_path  = "confirmable_notification.yaml"
  blueprint_input = jsonencode({
    notify_device = "abc123"
    title         = "Garage door"
  })
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("homeassistant_script.from_blueprint", "id"),
					resource.TestCheckResourceAttr("homeassistant_script.from_blueprint", "alias", "Confirmable Notification"),
					resource.TestCheckResourceAttr("homeassistant_script.from_blueprint", "blueprint_path", "confirmable_notification.yaml"),
					resource.TestCheckResourceAttr("homeassistant_script.from_blueprint", "blueprint_input", `{"notify_device":"abc123","title":"Garage door"}`),
					resource.TestCheckNoResourceAttr("homeassistant_script.from_blueprint", "sequence"),
				),
			},
		},
	})
}

func TestScriptResource_blueprintConflictsWithSequence(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "bad" {
  alias          = "Bad"
  blueprint_path = "confirmable_notification.yaml"
  sequence       = jsonencode([{ service = "notify.notify" }])
}
`,
				ExpectError: regexp.MustCompile(`Conflicting configuration`),
			},
		},
	})
}

func TestScriptResource_requiresSequenceWithoutBlueprint(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "bad" {
  alias = "Bad"
}
`,
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
		},
	})
}

func TestScriptResource_update(t *testing.T) {
	srv := testutil.NewMockHAServer(t, testutil.MockServerConfig{Token: "test-token"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "notify" {
  alias    = "Old Alias"
  sequence = jsonencode([{ service = "notify.notify" }])
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_script.notify", "alias", "Old Alias"),
			},
			{
				Config: providerConfig(srv.URL, "test-token") + `
resource "homeassistant_script" "notify" {
  alias    = "New Alias"
  sequence = jsonencode([{ service = "light.turn_on" }])
}
`,
				Check: resource.TestCheckResourceAttr("homeassistant_script.notify", "alias", "New Alias"),
			},
		},
	})
}
