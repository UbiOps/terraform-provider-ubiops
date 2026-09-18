// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &RoleResource{}
	_ resource.ResourceWithImportState = &RoleResource{}
	_ resource.ResourceWithConfigure   = &RoleResource{}
)

// NewRoleResource returns a new role resource.
func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// RoleResource manages a UbiOps role.
type RoleResource struct {
	client *client.UbiOpsClient
}

// RoleResourceModel maps the role schema to Go types.
type RoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectName types.String `tfsdk:"project_name"`
	Name        types.String `tfsdk:"name"`
	Permissions types.List   `tfsdk:"permissions"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps role with custom permissions.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the created role (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project this role belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the created role",
				Required:            true,
			},
			"permissions": schema.ListAttribute{
				MarkdownDescription: "A list of permissions which the role contains",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":        data.Name.ValueString(),
		"permissions": permissions,
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/roles", url.PathEscape(projectName)), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating role", err.Error())
		return
	}

	readRoleResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created role", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/roles/%s", url.PathEscape(projectName), url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}

	readRoleResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleResourceModel
	var state RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "name", plan.Name, state.Name)

	if !plan.Permissions.Equal(state.Permissions) {
		var permissions []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &permissions, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["permissions"] = permissions
	}

	projectName := state.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/roles/%s", url.PathEscape(projectName), url.PathEscape(state.Name.ValueString())), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating role", err.Error())
		return
	}

	readRoleResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated role", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/roles/%s", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString())))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting role", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted role", map[string]any{"name": data.Name.ValueString()})
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readRoleResult maps the API response to the Terraform model.
func readRoleResult(ctx context.Context, result map[string]any, data *RoleResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}

	// Permissions.
	if v, ok := result["permissions"]; ok && v != nil {
		if perms, ok := v.([]any); ok {
			vals := make([]string, 0, len(perms))
			for _, p := range perms {
				if s, ok := p.(string); ok {
					vals = append(vals, s)
				}
			}
			l, _ := types.ListValueFrom(ctx, types.StringType, vals)
			data.Permissions = l
		} else {
			data.Permissions = types.ListNull(types.StringType)
		}
	} else {
		data.Permissions = types.ListNull(types.StringType)
	}
}
