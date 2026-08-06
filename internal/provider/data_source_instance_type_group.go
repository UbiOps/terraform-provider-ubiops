// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &InstanceTypeGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &InstanceTypeGroupDataSource{}
)

// NewInstanceTypeGroupDataSource returns a new instance type group data source.
func NewInstanceTypeGroupDataSource() datasource.DataSource {
	return &InstanceTypeGroupDataSource{}
}

// InstanceTypeGroupDataSource reads a UbiOps instance type group.
type InstanceTypeGroupDataSource struct {
	client *client.UbiOpsClient
}

// InstanceTypeGroupDataSourceModel maps the data source schema to Go types.
type InstanceTypeGroupDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectName       types.String `tfsdk:"project_name"`
	Name              types.String `tfsdk:"name"`
	InstanceTypesJSON types.String `tfsdk:"instance_types_json"`
	TimeCreated       types.String `tfsdk:"time_created"`
}

func (d *InstanceTypeGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_type_group"
}

func (d *InstanceTypeGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get information about an existing UbiOps instance type group.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the instance type group (UUID)",
				Required:            true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the instance type group",
				Computed:            true,
			},
			"instance_types_json": schema.StringAttribute{
				MarkdownDescription: "A list of instance types that are in this group",
				Computed:            true,
			},
			"time_created": schema.StringAttribute{
				MarkdownDescription: "The date when the instance type group was created",
				Computed:            true,
			},
		},
	}
}

func (d *InstanceTypeGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceTypeGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceTypeGroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := d.client.Get(ctx, fmt.Sprintf("/projects/%s/instance-type-groups/%s", data.ProjectName.ValueString(), data.ID.ValueString()), &result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading instance type group", err.Error())
		return
	}

	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["time_created"].(string); ok {
		data.TimeCreated = types.StringValue(v)
	}

	if v, ok := result["instance_types"]; ok && v != nil {
		b, err := json.Marshal(v)
		if err == nil {
			data.InstanceTypesJSON = types.StringValue(string(b))
		}
	} else {
		data.InstanceTypesJSON = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
