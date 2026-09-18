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
	_ datasource.DataSource              = &EnvironmentDataSource{}
	_ datasource.DataSourceWithConfigure = &EnvironmentDataSource{}
)

// NewEnvironmentDataSource returns a new environment data source.
func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

// EnvironmentDataSource reads a UbiOps environment.
type EnvironmentDataSource struct {
	client *client.UbiOpsClient
}

// EnvironmentDataSourceModel maps the data source schema to Go types.
type EnvironmentDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ProjectName           types.String `tfsdk:"project_name"`
	Name                  types.String `tfsdk:"name"`
	DisplayName           types.String `tfsdk:"display_name"`
	BaseEnvironment       types.String `tfsdk:"base_environment"`
	Description           types.String `tfsdk:"description"`
	SupportsRequestFormat types.Bool   `tfsdk:"supports_request_format"`
	Labels                types.Map    `tfsdk:"labels"`
	CreationDate          types.String `tfsdk:"creation_date"`
	LastUpdated           types.String `tfsdk:"last_updated"`
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get information about an existing UbiOps environment.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the environment (UUID)",
				Computed:            true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the environment",
				Computed:            true,
			},
			"base_environment": schema.StringAttribute{
				MarkdownDescription: "Base environment name this environment is based on",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the environment",
				Computed:            true,
			},
			"supports_request_format": schema.BoolAttribute{
				MarkdownDescription: "Whether the environment supports UbiOps's structured request format (queuing, autoscaling, scheduled requests). Must match the `supports_request_format` of any deployment using this environment, or version creation fails.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the environment was created",
				Computed:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the environment was last updated",
				Computed:            true,
			},
		},
	}
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := d.client.Get(ctx, fmt.Sprintf("/projects/%s/environments/%s", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment", err.Error())
		return
	}

	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["display_name"].(string); ok {
		data.DisplayName = types.StringValue(v)
	}
	if v, ok := result["base_environment"].(string); ok {
		data.BaseEnvironment = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["supports_request_format"].(bool); ok {
		data.SupportsRequestFormat = types.BoolValue(v)
	}
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
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
