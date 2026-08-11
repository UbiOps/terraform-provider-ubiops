// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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
	_ resource.Resource                = &ServiceResource{}
	_ resource.ResourceWithImportState = &ServiceResource{}
	_ resource.ResourceWithConfigure   = &ServiceResource{}
)

// NewServiceResource returns a new service resource.
func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource manages a UbiOps service.
type ServiceResource struct {
	client *client.UbiOpsClient
}

// ServiceResourceModel maps the service schema to Go types.
type ServiceResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	ProjectName             types.String `tfsdk:"project_name"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Deployment              types.String `tfsdk:"deployment"`
	Version                 types.String `tfsdk:"version"`
	Port                    types.Int64  `tfsdk:"port"`
	AuthenticationRequired  types.Bool   `tfsdk:"authentication_required"`
	AuthMethodTokenEnabled  types.Bool   `tfsdk:"authentication_method_token_enabled"`
	RateLimitToken          types.Int64  `tfsdk:"rate_limit_token"`
	RequestLoggingExclPaths types.String `tfsdk:"request_logging_excluded_paths"`
	Endpoint                types.String `tfsdk:"endpoint"`
	Labels                  types.Map    `tfsdk:"labels"`
	TimeCreated             types.String `tfsdk:"time_created"`
	TimeUpdated             types.String `tfsdk:"time_updated"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps service (long-running deployment with a persistent endpoint).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the service (UUID)",
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
				MarkdownDescription: "Name of the service",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the service",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"deployment": schema.StringAttribute{
				MarkdownDescription: "Deployment of the service",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of the service. If null, the default version of the deployment is used.",
				Optional:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Port in the instances that are exposed",
				Required:            true,
			},
			"authentication_required": schema.BoolAttribute{
				MarkdownDescription: "Whether authentication is required on this service",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"authentication_method_token_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether authentication with a token is enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"rate_limit_token": schema.Int64Attribute{
				MarkdownDescription: "Rate limit for the service per authentication token",
				Optional:            true,
			},
			"request_logging_excluded_paths": schema.StringAttribute{
				MarkdownDescription: "A regex to exclude paths when storing requests",
				Optional:            true,
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Service endpoint URL",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"time_created": schema.StringAttribute{
				MarkdownDescription: "The date when the service was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"time_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the service was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":       data.Name.ValueString(),
		"deployment": data.Deployment.ValueString(),
		"port":       data.Port.ValueInt64(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalString(body, "request_logging_excluded_paths", data.RequestLoggingExclPaths)
	setOptionalBool(body, "authentication_required", data.AuthenticationRequired)
	setOptionalBool(body, "authentication_method_token_enabled", data.AuthMethodTokenEnabled)
	setOptionalInt64(body, "rate_limit_token", data.RateLimitToken)

	if !data.Version.IsNull() && !data.Version.IsUnknown() {
		body["version"] = data.Version.ValueString()
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
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/services", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}

	readServiceResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created service", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/services/%s", projectName, data.Name.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}

	readServiceResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceResourceModel
	var state ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "name", plan.Name, state.Name)
	setChangedString(body, "description", plan.Description, state.Description)
	setChangedString(body, "deployment", plan.Deployment, state.Deployment)
	setChangedString(body, "request_logging_excluded_paths", plan.RequestLoggingExclPaths, state.RequestLoggingExclPaths)
	setChangedBool(body, "authentication_required", plan.AuthenticationRequired, state.AuthenticationRequired)
	setChangedBool(body, "authentication_method_token_enabled", plan.AuthMethodTokenEnabled, state.AuthMethodTokenEnabled)
	setChangedInt64(body, "port", plan.Port, state.Port)
	setChangedInt64(body, "rate_limit_token", plan.RateLimitToken, state.RateLimitToken)

	if !plan.Version.Equal(state.Version) {
		if plan.Version.IsNull() {
			body["version"] = nil
		} else {
			body["version"] = plan.Version.ValueString()
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
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/services/%s", projectName, state.Name.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating service", err.Error())
		return
	}

	readServiceResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated service", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/services/%s", data.ProjectName.ValueString(), data.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting service", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted service", map[string]any{"name": data.Name.ValueString()})
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readServiceResult maps the API response to the Terraform model.
func readServiceResult(ctx context.Context, result map[string]any, data *ServiceResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["deployment"].(string); ok {
		data.Deployment = types.StringValue(v)
	}
	if v, ok := result["time_created"].(string); ok {
		data.TimeCreated = types.StringValue(v)
	}
	if v, ok := result["time_updated"].(string); ok {
		data.TimeUpdated = types.StringValue(v)
	}
	if v, ok := result["request_logging_excluded_paths"].(string); ok {
		data.RequestLoggingExclPaths = types.StringValue(v)
	}

	readInt64Field(result, "port", &data.Port)
	readInt64Field(result, "rate_limit_token", &data.RateLimitToken)
	readBoolField(result, "authentication_required", &data.AuthenticationRequired)
	readBoolField(result, "authentication_method_token_enabled", &data.AuthMethodTokenEnabled)

	// Version can be null.
	if v, ok := result["version"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Version = types.StringValue(s)
		}
	} else {
		data.Version = types.StringNull()
	}

	// Endpoint can be null.
	if v, ok := result["endpoint"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Endpoint = types.StringValue(s)
		}
	} else {
		data.Endpoint = types.StringNull()
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
