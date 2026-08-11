// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &RequestScheduleResource{}
	_ resource.ResourceWithImportState = &RequestScheduleResource{}
	_ resource.ResourceWithConfigure   = &RequestScheduleResource{}
)

// NewRequestScheduleResource returns a new request schedule resource.
func NewRequestScheduleResource() resource.Resource {
	return &RequestScheduleResource{}
}

// RequestScheduleResource manages a UbiOps request schedule.
type RequestScheduleResource struct {
	client *client.UbiOpsClient
}

// RequestScheduleResourceModel maps the request schedule schema to Go types.
type RequestScheduleResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectName     types.String `tfsdk:"project_name"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ObjectType      types.String `tfsdk:"object_type"`
	ObjectName      types.String `tfsdk:"object_name"`
	Version         types.String `tfsdk:"version"`
	Schedule        types.String `tfsdk:"schedule"`
	RequestDataJSON types.String `tfsdk:"request_data_json"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Timeout         types.Int64  `tfsdk:"timeout"`
	Labels          types.Map    `tfsdk:"labels"`
	CreationDate    types.String `tfsdk:"creation_date"`
}

func (r *RequestScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_request_schedule"
}

func (r *RequestScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps request schedule.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the schedule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the request",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the request schedule",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"object_type": schema.StringAttribute{
				MarkdownDescription: "Type of object for which the request is made",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_name": schema.StringAttribute{
				MarkdownDescription: "Name of deployment/pipeline for which the request schedule is made",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Name of version for which the request schedule is made",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schedule": schema.StringAttribute{
				MarkdownDescription: "Schedule in crontab format",
				Required:            true,
			},
			"request_data_json": schema.StringAttribute{
				MarkdownDescription: "Input data for the request schedule",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Boolean value indicating whether the request schedule is enabled or disabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout of the request in seconds",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the request schedule was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RequestScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *RequestScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RequestScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":        data.Name.ValueString(),
		"object_type": data.ObjectType.ValueString(),
		"object_name": data.ObjectName.ValueString(),
		"schedule":    data.Schedule.ValueString(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalBool(body, "enabled", data.Enabled)
	setOptionalInt64(body, "timeout", data.Timeout)

	if !data.Version.IsNull() && !data.Version.IsUnknown() {
		body["version"] = data.Version.ValueString()
	}

	if !data.RequestDataJSON.IsNull() && !data.RequestDataJSON.IsUnknown() {
		var requestData any
		if err := json.Unmarshal([]byte(data.RequestDataJSON.ValueString()), &requestData); err != nil {
			resp.Diagnostics.AddError("Invalid request_data_json", err.Error())
			return
		}
		body["request_data"] = requestData
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
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/schedules", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request schedule", err.Error())
		return
	}

	readRequestScheduleResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created request schedule", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequestScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RequestScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/schedules/%s", projectName, data.Name.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading request schedule", err.Error())
		return
	}

	readRequestScheduleResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RequestScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RequestScheduleResourceModel
	var state RequestScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "name", plan.Name, state.Name)
	setChangedString(body, "description", plan.Description, state.Description)
	setChangedString(body, "schedule", plan.Schedule, state.Schedule)
	setChangedBool(body, "enabled", plan.Enabled, state.Enabled)

	if !plan.Timeout.Equal(state.Timeout) {
		if plan.Timeout.IsNull() {
			body["timeout"] = nil
		} else {
			body["timeout"] = plan.Timeout.ValueInt64()
		}
	}

	if !plan.RequestDataJSON.Equal(state.RequestDataJSON) {
		if plan.RequestDataJSON.IsNull() {
			body["request_data"] = nil
		} else {
			var requestData any
			if err := json.Unmarshal([]byte(plan.RequestDataJSON.ValueString()), &requestData); err != nil {
				resp.Diagnostics.AddError("Invalid request_data_json", err.Error())
				return
			}
			body["request_data"] = requestData
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
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/schedules/%s", projectName, state.Name.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating request schedule", err.Error())
		return
	}

	readRequestScheduleResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated request schedule", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RequestScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RequestScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/schedules/%s", data.ProjectName.ValueString(), data.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting request schedule", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted request schedule", map[string]any{"name": data.Name.ValueString()})
}

func (r *RequestScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readRequestScheduleResult maps the API response to the Terraform model.
func readRequestScheduleResult(ctx context.Context, result map[string]any, data *RequestScheduleResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["object_type"].(string); ok {
		data.ObjectType = types.StringValue(v)
	}
	if v, ok := result["object_name"].(string); ok {
		data.ObjectName = types.StringValue(v)
	}
	if v, ok := result["schedule"].(string); ok {
		data.Schedule = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}

	readBoolField(result, "enabled", &data.Enabled)

	// Timeout comes back as string from GET but int from POST; handle both.
	if v, ok := result["timeout"]; ok && v != nil {
		switch t := v.(type) {
		case float64:
			data.Timeout = types.Int64Value(int64(t))
		case string:
			// Ignore string representation; keep existing state.
		}
	}

	// Version can be null.
	if v, ok := result["version"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Version = types.StringValue(s)
		}
	} else {
		data.Version = types.StringNull()
	}

	// Request data.
	if v, ok := result["request_data"]; ok && v != nil {
		b, err := json.Marshal(v)
		if err == nil {
			data.RequestDataJSON = types.StringValue(string(b))
		}
	} else {
		data.RequestDataJSON = types.StringNull()
	}

	// Labels.
	// Labels — store null when empty so plan null stays null.
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
