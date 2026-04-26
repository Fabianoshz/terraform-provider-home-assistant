package provider_test

import (
	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func providerFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"homeassistant": providerserver.NewProtocol6WithError(provider.New()),
	}
}
