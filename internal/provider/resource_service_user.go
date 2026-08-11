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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ServiceUserResource{}
	_ resource.ResourceWithImportState = &ServiceUserResource{}
	_ resource.ResourceWithConfigure   = &ServiceUserResource{}
)

// NewServiceUserResource returns a new service user resource.
func NewServiceUserResource() resource.Resource {
	return &ServiceUserResource{}
}

// ServiceUserResource manages a UbiOps service user.
type ServiceUserResource struct {
	client *client.UbiOpsClient
}

// ServiceUserResourceModel maps the service user schema to Go types.
type ServiceUserResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectName  types.String `tfsdk:"project_name"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Email        types.String `tfsdk:"email"`
	ExpiryDate   types.String `tfsdk:"expiry_date"`
	CreationDate types.String `tfsdk:"creation_date"`
}

func (r *ServiceUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_user"
}

func (r *ServiceUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps service user.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the service user (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project this service user belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service user",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the service user",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email of the service user",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expiry_date": schema.StringAttribute{
				MarkdownDescription: "Date when the service user account will expire (UTC)",
				Optional:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "Date when the service user was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ServiceUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *ServiceUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setOptionalString(body, "name", data.Name)
	setOptionalString(body, "description", data.Description)
	setOptionalString(body, "expiry_date", data.ExpiryDate)

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/service-users", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service user", err.Error())
		return
	}

	readServiceUserResult(result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created service user", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/service-users/%s", projectName, data.ID.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service user", err.Error())
		return
	}

	readServiceUserResult(result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceUserResourceModel
	var state ServiceUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "name", plan.Name, state.Name)
	setChangedString(body, "description", plan.Description, state.Description)

	if !plan.ExpiryDate.Equal(state.ExpiryDate) {
		if plan.ExpiryDate.IsNull() {
			body["expiry_date"] = nil
		} else {
			body["expiry_date"] = plan.ExpiryDate.ValueString()
		}
	}

	projectName := state.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/service-users/%s", projectName, state.ID.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating service user", err.Error())
		return
	}

	readServiceUserResult(result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated service user", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServiceUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/service-users/%s", data.ProjectName.ValueString(), data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting service user", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted service user", map[string]any{"id": data.ID.ValueString()})
}

func (r *ServiceUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "id", resp)
}

// readServiceUserResult maps the API response to the Terraform model.
func readServiceUserResult(result map[string]any, data *ServiceUserResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["email"].(string); ok {
		data.Email = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}

	// Expiry date can be null.
	if v, ok := result["expiry_date"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.ExpiryDate = types.StringValue(s)
		}
	} else {
		data.ExpiryDate = types.StringNull()
	}
}
