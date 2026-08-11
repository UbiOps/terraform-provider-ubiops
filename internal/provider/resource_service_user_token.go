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
	_ resource.Resource              = &ServiceUserTokenResource{}
	_ resource.ResourceWithConfigure = &ServiceUserTokenResource{}
)

// NewServiceUserTokenResource returns a new service user token resource.
func NewServiceUserTokenResource() resource.Resource {
	return &ServiceUserTokenResource{}
}

// ServiceUserTokenResource manages a UbiOps service user token.
type ServiceUserTokenResource struct {
	client *client.UbiOpsClient
}

// ServiceUserTokenResourceModel maps the service user token schema to Go types.
type ServiceUserTokenResourceModel struct {
	ProjectName   types.String `tfsdk:"project_name"`
	ServiceUserID types.String `tfsdk:"service_user_id"`
	Token         types.String `tfsdk:"token"`
}

func (r *ServiceUserTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_user_token"
}

func (r *ServiceUserTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps service user API token. The token value is only available at creation time.",

		Attributes: map[string]schema.Attribute{
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_user_id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The new API token for the service user",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ServiceUserTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *ServiceUserTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceUserTokenResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()
	serviceUserID := data.ServiceUserID.ValueString()

	// The API uses PUT to create/reset the token.
	var result map[string]any
	err := r.client.Do(ctx, "PUT", fmt.Sprintf("/projects/%s/service-users/%s/token", projectName, serviceUserID), map[string]any{}, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service user token", err.Error())
		return
	}

	if v, ok := result["token"].(string); ok {
		data.Token = types.StringValue(v)
	}

	tflog.Trace(ctx, "created service user token", map[string]any{"service_user_id": serviceUserID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceUserTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Token value is only available at creation time; confirm state as-is.
	var data ServiceUserTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceUserTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes require replace, so Update is never called.
	resp.Diagnostics.AddError("Unexpected update", "Service user tokens cannot be updated in-place.")
}

func (r *ServiceUserTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The API provides no dedicated delete for a token; creating a new one invalidates the old one.
	// Simply remove from state.
	tflog.Trace(ctx, "removed service user token from state")
}
