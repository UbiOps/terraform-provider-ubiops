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
	_ resource.Resource                = &RoleAssignmentResource{}
	_ resource.ResourceWithImportState = &RoleAssignmentResource{}
	_ resource.ResourceWithConfigure   = &RoleAssignmentResource{}
)

// NewRoleAssignmentResource returns a new role assignment resource.
func NewRoleAssignmentResource() resource.Resource {
	return &RoleAssignmentResource{}
}

// RoleAssignmentResource manages a UbiOps role assignment.
type RoleAssignmentResource struct {
	client *client.UbiOpsClient
}

// RoleAssignmentResourceModel maps the role assignment schema to Go types.
type RoleAssignmentResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectName  types.String `tfsdk:"project_name"`
	Role         types.String `tfsdk:"role"`
	Assignee     types.String `tfsdk:"assignee"`
	AssigneeType types.String `tfsdk:"assignee_type"`
	Resource     types.String `tfsdk:"resource"`
	ResourceType types.String `tfsdk:"resource_type"`
}

func (r *RoleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_assignment"
}

func (r *RoleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps role assignment.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the role assignment (UUID)",
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
			"role": schema.StringAttribute{
				MarkdownDescription: "Name of the role assigned to the user/object",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"assignee": schema.StringAttribute{
				MarkdownDescription: "UUID of the user or the name of the object",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"assignee_type": schema.StringAttribute{
				MarkdownDescription: "Type of the assignee",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource": schema.StringAttribute{
				MarkdownDescription: "Name of the object for which the role is assigned. Defaults to the project when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "Type of the object for which the role is assigned. Defaults to \"project\" when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *RoleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *RoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"role":          data.Role.ValueString(),
		"assignee":      data.Assignee.ValueString(),
		"assignee_type": data.AssigneeType.ValueString(),
	}

	setOptionalString(body, "resource", data.Resource)
	setOptionalString(body, "resource_type", data.ResourceType)

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/role-assignments", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating role assignment", err.Error())
		return
	}

	readRoleAssignmentResult(result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created role assignment", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/role-assignments/%s", projectName, data.ID.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading role assignment", err.Error())
		return
	}

	readRoleAssignmentResult(result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes require replace, so Update is never called.
	resp.Diagnostics.AddError("Unexpected update", "Role assignments cannot be updated in-place.")
}

func (r *RoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/role-assignments/%s", data.ProjectName.ValueString(), data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting role assignment", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted role assignment", map[string]any{"id": data.ID.ValueString()})
}

func (r *RoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "id", resp)
}

// readRoleAssignmentResult maps the API response to the Terraform model.
func readRoleAssignmentResult(result map[string]any, data *RoleAssignmentResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["role"].(string); ok {
		data.Role = types.StringValue(v)
	}
	if v, ok := result["assignee"].(string); ok {
		data.Assignee = types.StringValue(v)
	}
	if v, ok := result["assignee_type"].(string); ok {
		data.AssigneeType = types.StringValue(v)
	}
	if v, ok := result["resource"].(string); ok {
		data.Resource = types.StringValue(v)
	}
	if v, ok := result["resource_type"].(string); ok {
		data.ResourceType = types.StringValue(v)
	}
}
