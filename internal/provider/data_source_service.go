// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
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
	ID                      types.String `tfsdk:"id"`
	ProjectName             types.String `tfsdk:"project_name"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Deployment              types.String `tfsdk:"deployment"`
	Version                 types.String `tfsdk:"version"`
	Port                    types.Int64  `tfsdk:"port"`
	AuthenticationRequired  types.Bool   `tfsdk:"authentication_required"`
	AuthMethodTokenEnabled  types.Bool   `tfsdk:"authentication_method_token_enabled"`
	RateLimitToken          types.Int64  `tfsdk:"rate_limit_token"`
	RequestLoggingExclPaths types.String `tfsdk:"request_logging_excluded_paths"`
	Endpoint                types.String `tfsdk:"endpoint"`
	Labels                  types.Map    `tfsdk:"labels"`
	TimeCreated             types.String `tfsdk:"time_created"`
	TimeUpdated             types.String `tfsdk:"time_updated"`
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
			"authentication_required": schema.BoolAttribute{
				MarkdownDescription: "Whether authentication is required on this service",
				Computed:            true,
			},
			"authentication_method_token_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether authentication with a token is enabled",
				Computed:            true,
			},
			"rate_limit_token": schema.Int64Attribute{
				MarkdownDescription: "Rate limit for the service per authentication token",
				Computed:            true,
			},
			"request_logging_excluded_paths": schema.StringAttribute{
				MarkdownDescription: "A regex to exclude paths when storing requests",
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Endpoint of the service",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"time_created": schema.StringAttribute{
				MarkdownDescription: "The date when the service was created",
				Computed:            true,
			},
			"time_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the service was last updated",
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
	err := d.client.Get(ctx, fmt.Sprintf("/projects/%s/services/%s", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString())), &result)
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
	if v, ok := result["time_updated"].(string); ok {
		data.TimeUpdated = types.StringValue(v)
	}
	if v, ok := result["request_logging_excluded_paths"].(string); ok {
		data.RequestLoggingExclPaths = types.StringValue(v)
	} else {
		data.RequestLoggingExclPaths = types.StringNull()
	}

	readInt64Field(result, "port", &data.Port)
	readInt64Field(result, "rate_limit_token", &data.RateLimitToken)
	readBoolField(result, "authentication_required", &data.AuthenticationRequired)
	readBoolField(result, "authentication_method_token_enabled", &data.AuthMethodTokenEnabled)

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

	if v, ok := result["labels"]; ok && v != nil {
		if labelsMap, ok := v.(map[string]any); ok && len(labelsMap) > 0 {
			vals := make(map[string]string, len(labelsMap))
			for k, val := range labelsMap {
				if s, ok := val.(string); ok {
					vals[k] = s
				}
			}
			m, _ := types.MapValueFrom(ctx, types.StringType, vals)
			data.Labels = m
		} else {
			data.Labels = types.MapNull(types.StringType)
		}
	} else {
		data.Labels = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
