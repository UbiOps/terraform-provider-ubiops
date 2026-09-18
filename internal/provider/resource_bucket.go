// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &BucketResource{}
	_ resource.ResourceWithImportState = &BucketResource{}
	_ resource.ResourceWithConfigure   = &BucketResource{}
)

// NewBucketResource returns a new bucket resource.
func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

// BucketResource manages a UbiOps bucket.
type BucketResource struct {
	client *client.UbiOpsClient
}

// BucketResourceModel maps the bucket schema to Go types.
type BucketResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectName    types.String `tfsdk:"project_name"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	BucketProvider types.String `tfsdk:"bucket_provider"`
	Configuration  types.Map    `tfsdk:"configuration"`
	Credentials    types.Map    `tfsdk:"credentials"`
	Labels         types.Map    `tfsdk:"labels"`
	TTL            types.Int64  `tfsdk:"ttl"`
	CreationDate   types.String `tfsdk:"creation_date"`
	LastUpdated    types.String `tfsdk:"last_updated"`
}

func (r *BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps bucket for file storage.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the bucket (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project this bucket belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the bucket",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the bucket",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"bucket_provider": schema.StringAttribute{
				MarkdownDescription: "Provider of the bucket",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("ubiops"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("ubiops", "google_cloud_storage", "amazon_s3", "azure_blob_storage"),
				},
			},
			"configuration": schema.MapAttribute{
				MarkdownDescription: "Additional configuration details for the bucket",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"credentials": schema.MapAttribute{
				MarkdownDescription: "Credentials to connect to the bucket",
				Optional:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Time to live for the files in the bucket",
				Optional:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the bucket was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date and time the bucket was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": data.Name.ValueString(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalString(body, "provider", data.BucketProvider)
	setOptionalInt64(body, "ttl", data.TTL)

	if !data.Configuration.IsNull() {
		config := make(map[string]string)
		resp.Diagnostics.Append(data.Configuration.ElementsAs(ctx, &config, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["configuration"] = config
	}

	if !data.Credentials.IsNull() {
		creds := make(map[string]string)
		resp.Diagnostics.Append(data.Credentials.ElementsAs(ctx, &creds, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["credentials"] = creds
	}

	if !data.Labels.IsNull() {
		labels := make(map[string]string)
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/buckets", url.PathEscape(projectName)), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating bucket", err.Error())
		return
	}

	readBucketResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created bucket", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/buckets/%s", url.PathEscape(projectName), url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading bucket", err.Error())
		return
	}

	readBucketResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketResourceModel
	var state BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "description", plan.Description, state.Description)

	if !plan.TTL.Equal(state.TTL) {
		if plan.TTL.IsNull() {
			body["ttl"] = nil
		} else {
			body["ttl"] = plan.TTL.ValueInt64()
		}
	}

	if !plan.Configuration.Equal(state.Configuration) {
		if plan.Configuration.IsNull() {
			body["configuration"] = map[string]string{}
		} else {
			config := make(map[string]string)
			resp.Diagnostics.Append(plan.Configuration.ElementsAs(ctx, &config, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["configuration"] = config
		}
	}

	if !plan.Credentials.Equal(state.Credentials) {
		if plan.Credentials.IsNull() {
			body["credentials"] = map[string]string{}
		} else {
			creds := make(map[string]string)
			resp.Diagnostics.Append(plan.Credentials.ElementsAs(ctx, &creds, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["credentials"] = creds
		}
	}

	if !plan.Labels.Equal(state.Labels) {
		if plan.Labels.IsNull() {
			body["labels"] = map[string]string{}
		} else {
			labels := make(map[string]string)
			resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["labels"] = labels
		}
	}

	projectName := state.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/buckets/%s", url.PathEscape(projectName), url.PathEscape(state.Name.ValueString())), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating bucket", err.Error())
		return
	}

	readBucketResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated bucket", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/buckets/%s", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString())))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting bucket", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted bucket", map[string]any{"name": data.Name.ValueString()})
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readBucketResult maps the API response to the Terraform model.
func readBucketResult(ctx context.Context, result map[string]any, data *BucketResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
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

	// Configuration - read from API but credentials are write-only.
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

	// Labels.
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
}
