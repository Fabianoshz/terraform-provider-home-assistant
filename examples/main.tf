terraform {
  required_providers {
    homeassistant = {
      source = "registry.terraform.io/Fabianoshz/home-assistant"
    }
  }
}

provider "homeassistant" {
  # url and token read from HOMEASSISTANT_URL and HOMEASSISTANT_TOKEN env vars
}

# data "homeassistant_devices" "all" {}

# data "homeassistant_devices" "filtered" {
#   filter {
#     source = "connections"
#     type   = "mac"
#     value  = "e4:65:b8:cf:5d:98"
#   }
# }

# output "all_devices" {
#   value = data.homeassistant_devices.all.devices
# }

# output "filtered_devices" {
#   value = data.homeassistant_devices.filtered.devices
# }

# data "homeassistant_entities" "all" {}

# output "all_entities" {
#   value = data.homeassistant_entities.all.entities
# }

resource "homeassistant_area" "kitchen_2" {
  name = "Kitchen 2"
  icon = "mdi:countertop"
}

data "homeassistant_devices" "kitchen_sonoff" {
  filter {
    source = "connections"
    type   = "mac"
    value  = "e4:65:b8:cf:5d:98"
  }
}

resource "homeassistant_device" "kitchen_sonoff" {
  device_id = data.homeassistant_devices.kitchen_sonoff.devices[0].id
  area_id   = homeassistant_area.kitchen_2.area_id
}

