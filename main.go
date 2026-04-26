package main

import (
	"context"
	"log"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/Fabianoshz/homeassistant",
	}

	err := providerserver.Serve(context.Background(), provider.New, opts)
	if err != nil {
		log.Fatal(err)
	}
}
