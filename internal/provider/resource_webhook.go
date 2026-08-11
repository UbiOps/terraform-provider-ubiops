// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &WebhookResource{}
	_ resource.ResourceWithImportState = &WebhookResource{}
	_ resource.ResourceWithConfigure   = &WebhookResource{}
)

// NewWebhookResource returns a new webhook resource.
func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

// WebhookResource manages a UbiOps webhook.
type WebhookResource struct {
	client *client.UbiOpsClient
}

// WebhookResourceModel maps the webhook schema to Go types.
type WebhookResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProjectName   types.String `tfsdk:"project_name"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	URL           types.String `tfsdk:"url"`
	Event         types.String `tfsdk:"event"`
	ObjectType    types.String `tfsdk:"object_type"`
	ObjectName    types.String `tfsdk:"object_name"`
	Version       types.String `tfsdk:"version"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Retry         types.Bool   `tfsdk:"retry"`
	IncludeResult types.Bool   `tfsdk:"include_result"`
	Timeout       types.Int64  `tfsdk:"timeout"`
	HeadersJSON   types.String `tfsdk:"headers_json"`
	Labels        types.Map    `tfsdk:"labels"`
	CreationDate  types.String `tfsdk:"creation_date"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

func (r *WebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps webhook.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the webhook (UUID)",
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
				MarkdownDescription: "Name of the webhook",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the webhook",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Callback url to send event payloads to",
				Required:            true,
			},
			"event": schema.StringAttribute{
				MarkdownDescription: "Event that triggers the webhook",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"deployment_request_started", "deployment_request_completed",
						"deployment_request_failed", "deployment_request_finished",
						"pipeline_request_started", "pipeline_request_completed",
						"pipeline_request_failed", "pipeline_request_finished",
					),
				},
			},
			"object_type": schema.StringAttribute{
				MarkdownDescription: "Type of object for which the webhook is created",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_name": schema.StringAttribute{
				MarkdownDescription: "Name of deployment or pipeline for which the webhook is created",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Name of deployment/pipeline version for which the webhook is created",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Boolean value indicating whether the webhook is enabled or disabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"retry": schema.BoolAttribute{
				MarkdownDescription: "Boolean value indicating whether the calls to webhook should be retried if they fail",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"include_result": schema.BoolAttribute{
				MarkdownDescription: "Boolean indicating whether the result of a request should be included in the webhook call. It defaults to False.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds on the call to webhook",
				Optional:            true,
			},
			"headers_json": schema.StringAttribute{
				MarkdownDescription: "Request headers to use when calling the webhook",
				Optional:            true,
				Sensitive:           true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date and time the webhook was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date and time the webhook was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *WebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":        data.Name.ValueString(),
		"url":         data.URL.ValueString(),
		"event":       data.Event.ValueString(),
		"object_type": data.ObjectType.ValueString(),
		"object_name": data.ObjectName.ValueString(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalBool(body, "enabled", data.Enabled)
	setOptionalBool(body, "retry", data.Retry)
	setOptionalBool(body, "include_result", data.IncludeResult)
	setOptionalInt64(body, "timeout", data.Timeout)

	if !data.Version.IsNull() && !data.Version.IsUnknown() {
		body["version"] = data.Version.ValueString()
	}

	if !data.HeadersJSON.IsNull() && !data.HeadersJSON.IsUnknown() {
		var headers []any
		if err := json.Unmarshal([]byte(data.HeadersJSON.ValueString()), &headers); err != nil {
			resp.Diagnostics.AddError("Invalid headers_json", err.Error())
			return
		}
		body["headers"] = headers
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
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/webhooks", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}

	readWebhookResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created webhook", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/webhooks/%s", projectName, data.Name.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}

	readWebhookResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookResourceModel
	var state WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"url": plan.URL.ValueString(),
	}

	setChangedString(body, "name", plan.Name, state.Name)
	setChangedString(body, "description", plan.Description, state.Description)
	setChangedString(body, "event", plan.Event, state.Event)
	setChangedBool(body, "enabled", plan.Enabled, state.Enabled)
	setChangedBool(body, "retry", plan.Retry, state.Retry)
	setChangedBool(body, "include_result", plan.IncludeResult, state.IncludeResult)

	if !plan.Timeout.Equal(state.Timeout) {
		if plan.Timeout.IsNull() {
			body["timeout"] = nil
		} else {
			body["timeout"] = plan.Timeout.ValueInt64()
		}
	}

	if !plan.HeadersJSON.Equal(state.HeadersJSON) {
		if plan.HeadersJSON.IsNull() {
			body["headers"] = []any{}
		} else {
			var headers []any
			if err := json.Unmarshal([]byte(plan.HeadersJSON.ValueString()), &headers); err != nil {
				resp.Diagnostics.AddError("Invalid headers_json", err.Error())
				return
			}
			body["headers"] = headers
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
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/webhooks/%s", projectName, state.Name.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", err.Error())
		return
	}

	readWebhookResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated webhook", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/webhooks/%s", data.ProjectName.ValueString(), data.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted webhook", map[string]any{"name": data.Name.ValueString()})
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readWebhookResult maps the API response to the Terraform model.
func readWebhookResult(ctx context.Context, result map[string]any, data *WebhookResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["url"].(string); ok {
		data.URL = types.StringValue(v)
	}
	if v, ok := result["event"].(string); ok {
		data.Event = types.StringValue(v)
	}
	if v, ok := result["object_type"].(string); ok {
		data.ObjectType = types.StringValue(v)
	}
	if v, ok := result["object_name"].(string); ok {
		data.ObjectName = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
	}

	readBoolField(result, "enabled", &data.Enabled)
	readBoolField(result, "retry", &data.Retry)
	readBoolField(result, "include_result", &data.IncludeResult)
	readInt64Field(result, "timeout", &data.Timeout)

	// Version can be null.
	if v, ok := result["version"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Version = types.StringValue(s)
		}
	} else {
		data.Version = types.StringNull()
	}

	// Headers from API are read-only; keep existing state for the sensitive headers_json.

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
