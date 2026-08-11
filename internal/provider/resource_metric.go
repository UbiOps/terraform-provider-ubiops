// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &MetricResource{}
	_ resource.ResourceWithImportState = &MetricResource{}
	_ resource.ResourceWithConfigure   = &MetricResource{}
)

// NewMetricResource returns a new metric resource.
func NewMetricResource() resource.Resource {
	return &MetricResource{}
}

// MetricResource manages a UbiOps custom metric.
type MetricResource struct {
	client *client.UbiOpsClient
}

// MetricResourceModel maps the metric schema to Go types.
type MetricResourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	ProjectName  types.String `tfsdk:"project_name"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	MetricType   types.String `tfsdk:"metric_type"`
	Unit         types.String `tfsdk:"unit"`
	Labels       types.List   `tfsdk:"labels"`
	Custom       types.Bool   `tfsdk:"custom"`
	CreationDate types.String `tfsdk:"creation_date"`
	LastUpdated  types.String `tfsdk:"last_updated"`
}

func (r *MetricResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric"
}

func (r *MetricResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom metric in a UbiOps project.",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Unique identifier for the metric",
				Computed:            true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the metric",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the metric",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"metric_type": schema.StringAttribute{
				MarkdownDescription: "Type of the metric",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("delta", "gauge"),
				},
			},
			"unit": schema.StringAttribute{
				MarkdownDescription: "Unit of the metric",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"labels": schema.ListAttribute{
				MarkdownDescription: "A list of labels that can be used to get data points containing the metric",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"custom": schema.BoolAttribute{
				MarkdownDescription: "A boolean indicating whether the metric is custom",
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the metric was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the metric was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *MetricResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *MetricResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MetricResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":        data.Name.ValueString(),
		"metric_type": data.MetricType.ValueString(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalString(body, "unit", data.Unit)

	if !data.Labels.IsNull() {
		var labels []string
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/metrics", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating metric", err.Error())
		return
	}

	readMetricResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created metric", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MetricResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MetricResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/metrics/%s", projectName, data.Name.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading metric", err.Error())
		return
	}

	readMetricResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MetricResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MetricResourceModel
	var state MetricResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "name", plan.Name, state.Name)
	setChangedString(body, "description", plan.Description, state.Description)
	setChangedString(body, "unit", plan.Unit, state.Unit)

	if !plan.Labels.Equal(state.Labels) {
		if plan.Labels.IsNull() {
			body["labels"] = []string{}
		} else {
			var labels []string
			resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["labels"] = labels
		}
	}

	projectName := state.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/metrics/%s", projectName, state.Name.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating metric", err.Error())
		return
	}

	readMetricResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated metric", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MetricResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MetricResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/metrics/%s", data.ProjectName.ValueString(), data.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting metric", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted metric", map[string]any{"name": data.Name.ValueString()})
}

func (r *MetricResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readMetricResult maps the API response to the Terraform model.
func readMetricResult(ctx context.Context, result map[string]any, data *MetricResourceModel) {
	if v, ok := result["id"].(float64); ok {
		data.ID = types.Int64Value(int64(v))
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["metric_type"].(string); ok {
		// The API returns lowercase ("gauge", "delta"); the schema validator requires
		// uppercase ("GAUGE", "DELTA"). Normalise to uppercase so state matches config.
		data.MetricType = types.StringValue(strings.ToUpper(v))
	}
	if v, ok := result["unit"].(string); ok {
		data.Unit = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
	}

	readBoolField(result, "custom", &data.Custom)

	// Labels.
	if v, ok := result["labels"]; ok && v != nil {
		if arr, ok := v.([]any); ok {
			vals := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					vals = append(vals, s)
				}
			}
			l, _ := types.ListValueFrom(ctx, types.StringType, vals)
			data.Labels = l
		} else {
			data.Labels = types.ListNull(types.StringType)
		}
	} else {
		data.Labels = types.ListNull(types.StringType)
	}
}
