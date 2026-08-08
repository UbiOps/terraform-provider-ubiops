// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &DeploymentEnvironmentVariableResource{}
	_ resource.ResourceWithImportState = &DeploymentEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure   = &DeploymentEnvironmentVariableResource{}
)

// NewDeploymentEnvironmentVariableResource returns a new deployment environment variable resource.
func NewDeploymentEnvironmentVariableResource() resource.Resource {
	return &DeploymentEnvironmentVariableResource{}
}

// DeploymentEnvironmentVariableResource manages a deployment-level environment variable.
type DeploymentEnvironmentVariableResource struct {
	client *client.UbiOpsClient
}

// DeploymentEnvironmentVariableResourceModel maps the schema to Go types.
type DeploymentEnvironmentVariableResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectName    types.String `tfsdk:"project_name"`
	DeploymentName types.String `tfsdk:"deployment_name"`
	Name           types.String `tfsdk:"name"`
	Value          types.String `tfsdk:"value"`
	Secret         types.Bool   `tfsdk:"secret"`
}

func (r *DeploymentEnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_environment_variable"
}

func (r *DeploymentEnvironmentVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a deployment-level environment variable in UbiOps.",

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
			},
		},
	}
}

func (r *DeploymentEnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *DeploymentEnvironmentVariableResource) basePath(projectName, deploymentName string) string {
	return fmt.Sprintf("/projects/%s/deployments/%s/environment-variables", projectName, deploymentName)
}

func (r *DeploymentEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentEnvironmentVariableResourceModel

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
	err := r.client.Post(ctx, r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &data.ID, &data.Name, &data.Secret)

	tflog.Trace(ctx, "created deployment environment variable", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("%s/%s", r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString()), data.ID.ValueString()), &result)
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

func (r *DeploymentEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentEnvironmentVariableResourceModel
	var state DeploymentEnvironmentVariableResourceModel

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

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("%s/%s", r.basePath(state.ProjectName.ValueString(), state.DeploymentName.ValueString()), state.ID.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &plan.ID, &plan.Name, &plan.Secret)

	tflog.Trace(ctx, "updated deployment environment variable", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DeploymentEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("%s/%s", r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString()), data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting environment variable", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted deployment environment variable", map[string]any{"name": data.Name.ValueString()})
}

func (r *DeploymentEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: project_name/deployment_name/env_var_id.
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: project_name/deployment_name/id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}
