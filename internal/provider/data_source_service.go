// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ServiceDataSource{}
	_ datasource.DataSourceWithConfigure = &ServiceDataSource{}
)

// NewServiceDataSource returns a new service data source.
func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

// ServiceDataSource reads a UbiOps service.
type ServiceDataSource struct {
	client *client.UbiOpsClient
}

// ServiceDataSourceModel maps the data source schema to Go types.
type ServiceDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectName types.String `tfsdk:"project_name"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Deployment  types.String `tfsdk:"deployment"`
	Version     types.String `tfsdk:"version"`
	Port        types.Int64  `tfsdk:"port"`
	Endpoint    types.String `tfsdk:"endpoint"`
	TimeCreated types.String `tfsdk:"time_created"`
}

func (d *ServiceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get information about an existing UbiOps service.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the service (UUID)",
				Computed:            true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the service",
				Computed:            true,
			},
			"deployment": schema.StringAttribute{
				MarkdownDescription: "Name of the deployment that backs the service",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of the deployment that backs the service",
				Computed:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Port on which the deployment listens for the service",
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Endpoint of the service",
				Computed:            true,
			},
			"time_created": schema.StringAttribute{
				MarkdownDescription: "The date when the service was created",
				Computed:            true,
			},
		},
	}
}

func (d *ServiceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.UbiOpsClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.UbiOpsClient, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := d.client.Get(ctx, fmt.Sprintf("/projects/%s/services/%s", data.ProjectName.ValueString(), data.Name.ValueString()), &result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}

	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["deployment"].(string); ok {
		data.Deployment = types.StringValue(v)
	}
	if v, ok := result["time_created"].(string); ok {
		data.TimeCreated = types.StringValue(v)
	}

	readInt64Field(result, "port", &data.Port)

	if v, ok := result["version"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Version = types.StringValue(s)
		}
	} else {
		data.Version = types.StringNull()
	}

	if v, ok := result["endpoint"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Endpoint = types.StringValue(s)
		}
	} else {
		data.Endpoint = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
