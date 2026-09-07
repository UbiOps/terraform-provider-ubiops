// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &ubiopsProvider{}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ubiopsProvider{
			version: version,
		}
	}
}

// ubiopsProvider is the provider implementation.
type ubiopsProvider struct {
	// version is the provider version on release, "dev" when built and run
	// locally, or "test" during acceptance testing.
	version string
}

// ubiopsProviderModel describes the provider data model.
type ubiopsProviderModel struct {
	APIToken  types.String `tfsdk:"api_token"`
	BaseURL   types.String `tfsdk:"base_url"`
	ResolveIP types.String `tfsdk:"resolve"`
}

// Metadata returns the provider type name.
func (p *ubiopsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ubiops"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *ubiopsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The UbiOps provider manages resources on the UbiOps platform for ML model serving and orchestration.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "API token for authentication. Can also be set via the `UBIOPS_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the UbiOps API. Defaults to `https://api.ubiops.com/v2.1`. Can also be set via the `UBIOPS_BASE_URL` environment variable.",
				Optional:            true,
			},
			"resolve": schema.StringAttribute{
				MarkdownDescription: "Pin the `base_url` host to a specific IP instead of resolving it via DNS (equivalent to curl --resolve), e.g. `34.6.140.180`. Useful when the DNS record for a self-hosted UbiOps instance hasn't propagated yet; remove this setting once that DNS record resolves correctly on its own.",
				Optional:            true,
			},
		},
	}
}

// Configure prepares a UbiOps API client for data sources and resources.
func (p *ubiopsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring UbiOps client")

	var config ubiopsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Prevent misconfiguration when attribute values are not yet known.
	if config.APIToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown UbiOps API Token",
			"The provider cannot create the UbiOps API client as there is an unknown configuration value for the API token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the UBIOPS_API_TOKEN environment variable.",
		)
	}

	if config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown UbiOps API Base URL",
			"The provider cannot create the UbiOps API client as there is an unknown configuration value for the base URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the UBIOPS_BASE_URL environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default to environment variables, then override with Terraform configuration.
	apiToken := os.Getenv("UBIOPS_API_TOKEN")
	baseURL := os.Getenv("UBIOPS_BASE_URL")

	if !config.APIToken.IsNull() {
		apiToken = config.APIToken.ValueString()
	}
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing UbiOps API Token",
			"The provider cannot create the UbiOps API client as there is a missing or empty value for the API token. "+
				"Set the api_token value in the configuration or use the UBIOPS_API_TOKEN environment variable.",
		)
		return
	}
	if baseURL == "" {
		baseURL = "https://api.ubiops.com/v2.1"
	}

	resolveIP := ""
	if !config.ResolveIP.IsNull() {
		resolveIP = config.ResolveIP.ValueString()
	}

	ctx = tflog.SetField(ctx, "ubiops_base_url", baseURL)
	ctx = tflog.SetField(ctx, "ubiops_api_token", apiToken)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "ubiops_api_token")

	tflog.Debug(ctx, "Creating UbiOps client")

	c := client.NewUbiOpsClient(baseURL, apiToken, p.version, resolveIP)

	resp.ResourceData = c
	resp.DataSourceData = c

	tflog.Info(ctx, "Configured UbiOps client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *ubiopsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
		NewDeploymentDataSource,
		NewEnvironmentDataSource,
		NewPipelineDataSource,
		NewBucketDataSource,
		NewServiceDataSource,
		NewInstanceTypeGroupDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *ubiopsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewDeploymentResource,
		NewDeploymentVersionResource,
		NewEnvironmentResource,
		NewProjectEnvironmentVariableResource,
		NewDeploymentEnvironmentVariableResource,
		NewDeploymentVersionEnvironmentVariableResource,
		NewBucketResource,
		NewPipelineResource,
		NewPipelineVersionResource,
		NewServiceUserResource,
		NewServiceUserTokenResource,
		NewRoleResource,
		NewRoleAssignmentResource,
		NewWebhookResource,
		NewRequestScheduleResource,
		NewServiceResource,
		NewInstanceTypeGroupResource,
		NewFileResource,
		NewOrganizationUserResource,
		NewProjectUserResource,
		NewMetricResource,
		NewOrganizationResource,
	}
}
