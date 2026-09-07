// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &DeploymentVersionEnvironmentVariableResource{}
	_ resource.ResourceWithImportState = &DeploymentVersionEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure   = &DeploymentVersionEnvironmentVariableResource{}
)

// NewDeploymentVersionEnvironmentVariableResource returns a new deployment version environment variable resource.
func NewDeploymentVersionEnvironmentVariableResource() resource.Resource {
	return &DeploymentVersionEnvironmentVariableResource{}
}

// DeploymentVersionEnvironmentVariableResource manages a deployment version-level environment variable.
type DeploymentVersionEnvironmentVariableResource struct {
	client *client.UbiOpsClient
}

// DeploymentVersionEnvironmentVariableResourceModel maps the schema to Go types.
type DeploymentVersionEnvironmentVariableResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectName    types.String `tfsdk:"project_name"`
	DeploymentName types.String `tfsdk:"deployment_name"`
	Version        types.String `tfsdk:"version"`
	Name           types.String `tfsdk:"name"`
	Value          types.String `tfsdk:"value"`
	Secret         types.Bool   `tfsdk:"secret"`
}

func (r *DeploymentVersionEnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_version_environment_variable"
}

func (r *DeploymentVersionEnvironmentVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a deployment version-level environment variable in UbiOps.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the environment variable",
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
			"deployment_name": schema.StringAttribute{
				MarkdownDescription: "The name of the deployment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the deployment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Variable name",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Variable value (will be null for secret variables)",
				Required:            true,
				Sensitive:           true,
			},
			"secret": schema.BoolAttribute{
				MarkdownDescription: "Boolean that indicates if this variable contains sensitive information",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DeploymentVersionEnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *DeploymentVersionEnvironmentVariableResource) basePath(projectName, deploymentName, version string) string {
	return fmt.Sprintf("/projects/%s/deployments/%s/versions/%s/environment-variables", url.PathEscape(projectName), url.PathEscape(deploymentName), url.PathEscape(version))
}

func (r *DeploymentVersionEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentVersionEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":   data.Name.ValueString(),
		"value":  data.Value.ValueString(),
		"secret": data.Secret.ValueBool(),
	}

	var result map[string]any
	err := r.client.Post(ctx, r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString(), data.Version.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &data.ID, &data.Name, &data.Secret)

	tflog.Trace(ctx, "created deployment version environment variable", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentVersionEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentVersionEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString(), data.Version.ValueString())

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("%s/%s", bp, data.ID.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &data.ID, &data.Name, &data.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentVersionEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentVersionEnvironmentVariableResourceModel
	var state DeploymentVersionEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.Value.Equal(state.Value) {
		body["value"] = plan.Value.ValueString()
	}
	if !plan.Secret.Equal(state.Secret) {
		body["secret"] = plan.Secret.ValueBool()
	}

	bp := r.basePath(state.ProjectName.ValueString(), state.DeploymentName.ValueString(), state.Version.ValueString())

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("%s/%s", bp, state.ID.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &plan.ID, &plan.Name, &plan.Secret)

	tflog.Trace(ctx, "updated deployment version environment variable", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DeploymentVersionEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentVersionEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString(), data.Version.ValueString())

	err := r.client.Delete(ctx, fmt.Sprintf("%s/%s", bp, data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting environment variable", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted deployment version environment variable", map[string]any{"name": data.Name.ValueString()})
}

func (r *DeploymentVersionEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: project_name/deployment_name/version/env_var_id.
	parts := strings.SplitN(req.ID, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: project_name/deployment_name/version/id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}
