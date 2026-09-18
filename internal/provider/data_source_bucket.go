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
	_ datasource.DataSource              = &BucketDataSource{}
	_ datasource.DataSourceWithConfigure = &BucketDataSource{}
)

// NewBucketDataSource returns a new bucket data source.
func NewBucketDataSource() datasource.DataSource {
	return &BucketDataSource{}
}

// BucketDataSource reads a UbiOps bucket.
type BucketDataSource struct {
	client *client.UbiOpsClient
}

// BucketDataSourceModel maps the data source schema to Go types.
type BucketDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectName    types.String `tfsdk:"project_name"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	BucketProvider types.String `tfsdk:"bucket_provider"`
	Configuration  types.Map    `tfsdk:"configuration"`
	Labels         types.Map    `tfsdk:"labels"`
	TTL            types.Int64  `tfsdk:"ttl"`
	CreationDate   types.String `tfsdk:"creation_date"`
	LastUpdated    types.String `tfsdk:"last_updated"`
}

func (d *BucketDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (d *BucketDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get information about an existing UbiOps bucket.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the bucket (UUID)",
				Computed:            true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the bucket",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the bucket",
				Computed:            true,
			},
			"bucket_provider": schema.StringAttribute{
				MarkdownDescription: "Provider of the bucket",
				Computed:            true,
			},
			"configuration": schema.MapAttribute{
				MarkdownDescription: "Additional configuration details for the bucket",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Time to live for the files in the bucket",
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the bucket was created",
				Computed:            true,
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date and time the bucket was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *BucketDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BucketDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := d.client.Get(ctx, fmt.Sprintf("/projects/%s/buckets/%s", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket", err.Error())
		return
	}

	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["provider"].(string); ok {
		data.BucketProvider = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
	}

	readInt64Field(result, "ttl", &data.TTL)

	if v, ok := result["configuration"]; ok && v != nil {
		if configMap, ok := v.(map[string]any); ok && len(configMap) > 0 {
			vals := make(map[string]string, len(configMap))
			for k, val := range configMap {
				if s, ok := val.(string); ok {
					vals[k] = s
				}
			}
			m, _ := types.MapValueFrom(ctx, types.StringType, vals)
			data.Configuration = m
		} else {
			data.Configuration = types.MapNull(types.StringType)
		}
	} else {
		data.Configuration = types.MapNull(types.StringType)
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
