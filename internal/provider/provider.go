package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sazabi/terraform-provider-onepassword/internal/opclient"
)

// Ensure the implementation satisfies the framework interface.
var _ provider.Provider = &onepasswordProvider{}

// onepasswordProvider is the provider implementation.
type onepasswordProvider struct {
	version string
}

// onepasswordProviderModel maps provider schema data.
type onepasswordProviderModel struct {
	Account types.String `tfsdk:"account"`
}

// New returns a provider factory for the given build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &onepasswordProvider{version: version}
	}
}

func (p *onepasswordProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "onepassword"
	resp.Version = p.version
}

func (p *onepasswordProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password vault structure and vault -> group access grants by wrapping the `op` CLI. " +
			"Authentication is whatever `op` resolves from the environment: a Service Account token (`OP_SERVICE_ACCOUNT_TOKEN`) " +
			"or an interactive admin `op signin` session. Granting a group access to an existing (adopted) vault requires the " +
			"human admin session, because a Service Account can only manage vaults it created.",
		Attributes: map[string]schema.Attribute{
			"account": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "The 1Password account to target (e.g. `my.1password.com`), passed as `op --account`. " +
					"Omit to let `op` use its default/only account.",
			},
		},
	}
}

func (p *onepasswordProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg onepasswordProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := opclient.New(cfg.Account.ValueString())
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *onepasswordProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVaultResource,
		NewVaultGroupAccessResource,
	}
}

func (p *onepasswordProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
