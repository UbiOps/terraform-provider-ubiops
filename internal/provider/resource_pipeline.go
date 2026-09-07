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
	_ resource.Resource                = &PipelineResource{}
	_ resource.ResourceWithImportState = &PipelineResource{}
	_ resource.ResourceWithConfigure   = &PipelineResource{}
)

// NewPipelineResource returns a new pipeline resource.
func NewPipelineResource() resource.Resource {
	return &PipelineResource{}
}

// PipelineResource manages a UbiOps pipeline.
type PipelineResource struct {
	client *client.UbiOpsClient
}

// PipelineResourceModel maps the pipeline schema to Go types.
type PipelineResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectName    types.String `tfsdk:"project_name"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	InputType      types.String `tfsdk:"input_type"`
	OutputType     types.String `tfsdk:"output_type"`
	InputFields    types.List   `tfsdk:"input_fields"`
	OutputFields   types.List   `tfsdk:"output_fields"`
	Labels         types.Map    `tfsdk:"labels"`
	DefaultVersion types.String `tfsdk:"default_version"`
	CreationDate   types.String `tfsdk:"creation_date"`
	LastUpdated    types.String `tfsdk:"last_updated"`
}

func (r *PipelineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (r *PipelineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps pipeline.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the pipeline (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project this pipeline belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the pipeline",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description for the pipeline",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"input_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pipeline input",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("structured", "plain"),
				},
			},
			"output_type": schema.StringAttribute{
				MarkdownDescription: "Type of the pipeline output",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("structured"),
				Validators: []validator.String{
					stringvalidator.OneOf("structured", "plain"),
				},
			},
			"input_fields": schema.ListNestedAttribute{
				MarkdownDescription: "A list of pipeline input fields with name, data_type and widget",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the field.",
							Required:            true,
						},
						"data_type": schema.StringAttribute{
							MarkdownDescription: "The data type of the field.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"int", "string", "double", "bool", "dict",
									"array_int", "array_double", "array_string",
									"file", "array_file",
								),
							},
						},
					},
				},
			},
			"output_fields": schema.ListNestedAttribute{
				MarkdownDescription: "A list of pipeline output fields with name, data_type and widget",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the field.",
							Required:            true,
						},
						"data_type": schema.StringAttribute{
							MarkdownDescription: "The data type of the field.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"int", "string", "double", "bool", "dict",
									"array_int", "array_double", "array_string",
									"file", "array_file",
								),
							},
						},
					},
				},
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"default_version": schema.StringAttribute{
				MarkdownDescription: "Default version of the pipeline. If it does not have a default version, it is not set.",
				Optional:            true,
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the pipeline was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the pipeline was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *PipelineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *PipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PipelineResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":       data.Name.ValueString(),
		"input_type": data.InputType.ValueString(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalString(body, "output_type", data.OutputType)

	if !data.InputFields.IsNull() {
		body["input_fields"] = fieldsToAPI(ctx, data.InputFields, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.OutputFields.IsNull() {
		body["output_fields"] = fieldsToAPI(ctx, data.OutputFields, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
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
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/pipelines", url.PathEscape(projectName)), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating pipeline", err.Error())
		return
	}

	readPipelineResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created pipeline", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PipelineResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/pipelines/%s", url.PathEscape(projectName), url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading pipeline", err.Error())
		return
	}

	readPipelineResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PipelineResourceModel
	var state PipelineResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "name", plan.Name, state.Name)
	setChangedString(body, "description", plan.Description, state.Description)
	setChangedString(body, "input_type", plan.InputType, state.InputType)
	setChangedString(body, "output_type", plan.OutputType, state.OutputType)
	setChangedString(body, "default_version", plan.DefaultVersion, state.DefaultVersion)

	if !plan.InputFields.Equal(state.InputFields) {
		if plan.InputFields.IsNull() {
			body["input_fields"] = []any{}
		} else {
			body["input_fields"] = fieldsToAPI(ctx, plan.InputFields, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	if !plan.OutputFields.Equal(state.OutputFields) {
		if plan.OutputFields.IsNull() {
			body["output_fields"] = []any{}
		} else {
			body["output_fields"] = fieldsToAPI(ctx, plan.OutputFields, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
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
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/pipelines/%s", url.PathEscape(projectName), url.PathEscape(state.Name.ValueString())), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating pipeline", err.Error())
		return
	}

	readPipelineResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated pipeline", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PipelineResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/pipelines/%s", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString())))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting pipeline", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted pipeline", map[string]any{"name": data.Name.ValueString()})
}

func (r *PipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readPipelineResult maps the API response to the Terraform model.
func readPipelineResult(ctx context.Context, result map[string]any, data *PipelineResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
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
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
	}

	// Default version can be null.
	if v, ok := result["default_version"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.DefaultVersion = types.StringValue(s)
		}
	} else {
		data.DefaultVersion = types.StringNull()
	}

	// Input/output fields - reuse deployment helpers.
	data.InputFields = fieldsFromAPI(result["input_fields"])
	data.OutputFields = fieldsFromAPI(result["output_fields"])

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
