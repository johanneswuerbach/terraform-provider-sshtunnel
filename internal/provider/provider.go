package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure SSHTunnelProvider satisfies various provider interfaces.
var _ provider.Provider = &SSHTunnelProvider{}
var _ provider.ProviderWithEphemeralResources = &SSHTunnelProvider{}

// SSHTunnelProvider defines the provider implementation.
type SSHTunnelProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type ProviderConfigData struct {
	Tracker *TunnelTracker
}

// SSHTunnelProviderModel describes the provider data model.
type SSHTunnelProviderModel struct{}

func (p *SSHTunnelProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sshtunnel"
	resp.Version = p.version
}

func (p *SSHTunnelProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The SSH Tunnel provider allow creating ephemeral SSH tunnels.",
	}
}

func (p *SSHTunnelProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data SSHTunnelProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	config := &ProviderConfigData{
		Tracker: NewTunnelTracker(),
	}

	resp.EphemeralResourceData = config
}

func (p *SSHTunnelProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *SSHTunnelProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *SSHTunnelProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewConnectionEphemeralResource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SSHTunnelProvider{
			version: version,
		}
	}
}
