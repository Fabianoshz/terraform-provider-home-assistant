package provider

import (
	"context"
	"os"

	"github.com/Fabianoshz/terraform-provider-homeassistant/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &HomeAssistantProvider{}

type HomeAssistantProvider struct{}

type HomeAssistantProviderModel struct {
	URL   types.String `tfsdk:"url"`
	Token types.String `tfsdk:"token"`
}

func New() provider.Provider {
	return &HomeAssistantProvider{}
}

func (p *HomeAssistantProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "homeassistant"
}

func (p *HomeAssistantProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for interacting with Home Assistant via its REST API.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "Home Assistant base URL (e.g. http://homeassistant.local:8123). " +
					"Can also be set via HOMEASSISTANT_URL environment variable.",
				Optional: true,
			},
			"token": schema.StringAttribute{
				Description: "Long-lived access token for Home Assistant. " +
					"Can also be set via HOMEASSISTANT_TOKEN environment variable.",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *HomeAssistantProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config HomeAssistantProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := os.Getenv("HOMEASSISTANT_URL")
	if !config.URL.IsNull() && !config.URL.IsUnknown() {
		url = config.URL.ValueString()
	}

	token := os.Getenv("HOMEASSISTANT_TOKEN")
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	}

	if url == "" {
		resp.Diagnostics.AddError(
			"Missing Home Assistant URL",
			"Set the url provider attribute or HOMEASSISTANT_URL environment variable.",
		)
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing Home Assistant Token",
			"Set the token provider attribute or HOMEASSISTANT_TOKEN environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(url, token)
	if err := c.Verify(ctx); err != nil {
		resp.Diagnostics.AddError("Failed to connect to Home Assistant", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *HomeAssistantProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAreasDataSource,
		NewAutomationsDataSource,
		NewDevicesDataSource,
		NewEntitiesDataSource,
	}
}

func (p *HomeAssistantProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAreaResource,
		NewAutomationResource,
		NewDeviceResource,
		NewEntityResource,
		NewFloorResource,
	}
}
