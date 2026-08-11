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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ProjectEnvironmentVariableResource{}
	_ resource.ResourceWithImportState = &ProjectEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure   = &ProjectEnvironmentVariableResource{}
)

// NewProjectEnvironmentVariableResource returns a new project environment variable resource.
func NewProjectEnvironmentVariableResource() resource.Resource {
	return &ProjectEnvironmentVariableResource{}
}

// ProjectEnvironmentVariableResource manages a project-level environment variable.
type ProjectEnvironmentVariableResource struct {
	client *client.UbiOpsClient
}

// ProjectEnvironmentVariableResourceModel maps the schema to Go types.
type ProjectEnvironmentVariableResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectName types.String `tfsdk:"project_name"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Secret      types.Bool   `tfsdk:"secret"`
}

func (r *ProjectEnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment_variable"
}

func (r *ProjectEnvironmentVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a project-level environment variable in UbiOps.",

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

func (r *ProjectEnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *ProjectEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectEnvironmentVariableResourceModel

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
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/environment-variables", data.ProjectName.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &data.ID, &data.Name, &data.Secret)

	tflog.Trace(ctx, "created project environment variable", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/environment-variables/%s", data.ProjectName.ValueString(), data.ID.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &data.ID, &data.Name, &data.Secret)
	// Value is not returned for secret variables, so keep the state value.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectEnvironmentVariableResourceModel
	var state ProjectEnvironmentVariableResourceModel

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
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/environment-variables/%s", state.ProjectName.ValueString(), state.ID.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment variable", err.Error())
		return
	}

	readEnvVarResult(result, &plan.ID, &plan.Name, &plan.Secret)

	tflog.Trace(ctx, "updated project environment variable", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/environment-variables/%s", data.ProjectName.ValueString(), data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting environment variable", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted project environment variable", map[string]any{"name": data.Name.ValueString()})
}

func (r *ProjectEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "id", resp)
}

// readEnvVarResult maps the common environment variable API fields.
func readEnvVarResult(result map[string]any, id *types.String, name *types.String, secret *types.Bool) {
	if v, ok := result["id"].(string); ok {
		*id = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		*name = types.StringValue(v)
	}
	if v, ok := result["secret"].(bool); ok {
		*secret = types.BoolValue(v)
	}
}
