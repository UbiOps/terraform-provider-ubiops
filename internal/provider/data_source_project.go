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
	_ datasource.DataSource              = &ProjectDataSource{}
	_ datasource.DataSourceWithConfigure = &ProjectDataSource{}
)

// NewProjectDataSource returns a new project data source.
func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

// ProjectDataSource reads a UbiOps project.
type ProjectDataSource struct {
	client *client.UbiOpsClient
}

// ProjectDataSourceModel maps the data source schema to Go types.
type ProjectDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	OrganizationName    types.String `tfsdk:"organization_name"`
	AdvancedPermissions types.Bool   `tfsdk:"advanced_permissions"`
	Credits             types.Number `tfsdk:"credits"`
	CORSOrigins         types.List   `tfsdk:"cors_origins"`
	Labels              types.Map    `tfsdk:"labels"`
	CreationDate        types.String `tfsdk:"creation_date"`
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get information about an existing UbiOps project.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the project (UUID)",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project",
				Required:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "Name of the organization in which the project is created",
				Computed:            true,
			},
			"advanced_permissions": schema.BoolAttribute{
				MarkdownDescription: "Boolean value indicating whether advanced permissions are enabled for the project",
				Computed:            true,
			},
			"credits": schema.NumberAttribute{
				MarkdownDescription: "Maximum usage of credits, calculated by multiplying the credit rate of a deployment instance type by the number of hours they are running",
				Computed:            true,
			},
			"cors_origins": schema.ListAttribute{
				MarkdownDescription: "List of origins from which the requests are allowed for the project",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the project was created",
				Computed:            true,
			},
		},
	}
}

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := d.client.Get(ctx, fmt.Sprintf("/projects/%s", url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading project", err.Error())
		return
	}

	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["organization_name"].(string); ok {
		data.OrganizationName = types.StringValue(v)
	}
	if v, ok := result["advanced_permissions"].(bool); ok {
		data.AdvancedPermissions = types.BoolValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}

	// Credits can be null.
	if v, ok := result["credits"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			data.Credits = types.NumberValue((&types.Number{}).ValueBigFloat().SetFloat64(f))
		}
	} else {
		data.Credits = types.NumberNull()
	}

	// CORS origins: treat empty list same as absent (null) so Optional-only fields stay consistent.
	if v, ok := result["cors_origins"]; ok && v != nil {
		if origins, ok := v.([]any); ok && len(origins) > 0 {
			vals := make([]types.String, 0, len(origins))
			for _, o := range origins {
				if s, ok := o.(string); ok {
					vals = append(vals, types.StringValue(s))
				}
			}
			list, _ := types.ListValueFrom(ctx, types.StringType, vals)
			data.CORSOrigins = list
		} else {
			data.CORSOrigins = types.ListNull(types.StringType)
		}
	} else {
		data.CORSOrigins = types.ListNull(types.StringType)
	}

	// Labels: treat empty map same as absent (null) so Optional-only fields stay consistent.
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
