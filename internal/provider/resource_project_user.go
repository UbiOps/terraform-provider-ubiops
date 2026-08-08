// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ProjectUserResource{}
	_ resource.ResourceWithImportState = &ProjectUserResource{}
	_ resource.ResourceWithConfigure   = &ProjectUserResource{}
)

// NewProjectUserResource returns a new project user resource.
func NewProjectUserResource() resource.Resource {
	return &ProjectUserResource{}
}

// ProjectUserResource manages a user assignment in a UbiOps project.
type ProjectUserResource struct {
	client *client.UbiOpsClient
}

// ProjectUserResourceModel maps the project user schema to Go types.
type ProjectUserResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectName types.String `tfsdk:"project_name"`
	UserID      types.String `tfsdk:"user_id"`
	Email       types.String `tfsdk:"email"`
	Name        types.String `tfsdk:"name"`
	Surname     types.String `tfsdk:"surname"`
}

func (r *ProjectUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_user"
}

func (r *ProjectUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a user to a UbiOps project.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the user (UUID)",
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
			"user_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the user",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email of the user",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the user",
				Computed:            true,
			},
			"surname": schema.StringAttribute{
				MarkdownDescription: "Surname of the user",
				Computed:            true,
			},
		},
	}
}

func (r *ProjectUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *ProjectUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"user_id": data.UserID.ValueString(),
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/users", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating project user", err.Error())
		return
	}

	readProjectUserResult(result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created project user", map[string]any{"user_id": data.UserID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/users/%s", projectName, data.ID.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project user", err.Error())
		return
	}

	readProjectUserResult(result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not supported — all fields require replacement.
func (r *ProjectUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Project user assignments cannot be updated, only replaced.")
}

func (r *ProjectUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/users/%s", data.ProjectName.ValueString(), data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting project user", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted project user", map[string]any{"user_id": data.UserID.ValueString()})
}

func (r *ProjectUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: project_name/user_id.
	parseImportID2(ctx, req.ID, "id", resp)
}

// readProjectUserResult maps the API response to the Terraform model.
func readProjectUserResult(result map[string]any, data *ProjectUserResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
		data.UserID = types.StringValue(v)
	}
	if v, ok := result["email"].(string); ok {
		data.Email = types.StringValue(v)
	}
	if v, ok := result["name"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Name = types.StringValue(s)
		}
	} else {
		data.Name = types.StringNull()
	}
	if v, ok := result["surname"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.Surname = types.StringValue(s)
		}
	} else {
		data.Surname = types.StringNull()
	}
}
