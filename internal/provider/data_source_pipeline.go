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
	_ datasource.DataSource              = &PipelineDataSource{}
	_ datasource.DataSourceWithConfigure = &PipelineDataSource{}
)

// NewPipelineDataSource returns a new pipeline data source.
func NewPipelineDataSource() datasource.DataSource {
	return &PipelineDataSource{}
}

// PipelineDataSource reads a UbiOps pipeline.
type PipelineDataSource struct {
	client *client.UbiOpsClient
}

// PipelineDataSourceModel maps the data source schema to Go types.
type PipelineDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectName    types.String `tfsdk:"project_name"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	InputType      types.String `tfsdk:"input_type"`
	OutputType     types.String `tfsdk:"output_type"`
	DefaultVersion types.String `tfsdk:"default_version"`
	CreationDate   types.String `tfsdk:"creation_date"`
}

func (d *PipelineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (d *PipelineDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get information about an existing UbiOps pipeline.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the pipeline (UUID)",
				Computed:            true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the pipeline",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the pipeline",
				Computed:            true,
			},
			"input_type": schema.StringAttribute{
				MarkdownDescription: "The type of the input of the pipeline",
				Computed:            true,
			},
			"output_type": schema.StringAttribute{
				MarkdownDescription: "The type of the output of the pipeline",
				Computed:            true,
			},
			"default_version": schema.StringAttribute{
				MarkdownDescription: "Default version of the pipeline",
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the pipeline was created",
				Computed:            true,
			},
		},
	}
}

func (d *PipelineDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PipelineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PipelineDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := d.client.Get(ctx, fmt.Sprintf("/projects/%s/pipelines/%s", data.ProjectName.ValueString(), data.Name.ValueString()), &result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading pipeline", err.Error())
		return
	}

	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["input_type"].(string); ok {
		data.InputType = types.StringValue(v)
	}
	if v, ok := result["output_type"].(string); ok {
		data.OutputType = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["default_version"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.DefaultVersion = types.StringValue(s)
		}
	} else {
		data.DefaultVersion = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
