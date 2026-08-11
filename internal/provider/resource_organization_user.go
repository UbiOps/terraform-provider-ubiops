// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &OrganizationUserResource{}
	_ resource.ResourceWithImportState = &OrganizationUserResource{}
	_ resource.ResourceWithConfigure   = &OrganizationUserResource{}
)

// NewOrganizationUserResource returns a new organization user resource.
func NewOrganizationUserResource() resource.Resource {
	return &OrganizationUserResource{}
}

// OrganizationUserResource manages a UbiOps organization user.
type OrganizationUserResource struct {
	client *client.UbiOpsClient
}

// OrganizationUserResourceModel maps the organization user schema to Go types.
type OrganizationUserResourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationName types.String `tfsdk:"organization_name"`
	Email            types.String `tfsdk:"email"`
	Admin            types.Bool   `tfsdk:"admin"`
	Status           types.String `tfsdk:"status"`
}

func (r *OrganizationUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_user"
}

func (r *OrganizationUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user in a UbiOps organization.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the user (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "The name of the organization.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email of the user",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"admin": schema.BoolAttribute{
				MarkdownDescription: "Boolean value indicating whether the user is an admin of the organization or not",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the user.",
				Computed:            true,
			},
		},
	}
}

func (r *OrganizationUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *OrganizationUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrganizationUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"email": data.Email.ValueString(),
	}

	setOptionalBool(body, "admin", data.Admin)

	orgName := data.OrganizationName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/organizations/%s/users", orgName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization user", err.Error())
		return
	}

	readOrganizationUserResult(result, &data)
	data.OrganizationName = types.StringValue(orgName)

	tflog.Trace(ctx, "created organization user", map[string]any{"email": data.Email.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrganizationUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := data.OrganizationName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/organizations/%s/users/%s", orgName, data.ID.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization user", err.Error())
		return
	}

	readOrganizationUserResult(result, &data)
	data.OrganizationName = types.StringValue(orgName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationUserResourceModel
	var state OrganizationUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedBool(body, "admin", plan.Admin, state.Admin)

	orgName := state.OrganizationName.ValueString()

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/organizations/%s/users/%s", orgName, state.ID.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization user", err.Error())
		return
	}

	readOrganizationUserResult(result, &plan)
	plan.OrganizationName = types.StringValue(orgName)

	tflog.Trace(ctx, "updated organization user", map[string]any{"email": plan.Email.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OrganizationUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/organizations/%s/users/%s", data.OrganizationName.ValueString(), data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting organization user", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted organization user", map[string]any{"email": data.Email.ValueString()})
}

func (r *OrganizationUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: organization_name/user_id.
	parseOrgImportID2(ctx, req.ID, &resp.Diagnostics, resp)
}

// parseOrgImportID2 parses an import ID in the format "org_name/user_id" for organization resources.
func parseOrgImportID2(ctx context.Context, id string, diags *diag.Diagnostics, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		diags.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: organization_name/id, got: %s", id),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// readOrganizationUserResult maps the API response to the Terraform model.
func readOrganizationUserResult(result map[string]any, data *OrganizationUserResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["email"].(string); ok {
		data.Email = types.StringValue(v)
	}
	if v, ok := result["status"].(string); ok {
		data.Status = types.StringValue(v)
	}
	readBoolField(result, "admin", &data.Admin)
}
