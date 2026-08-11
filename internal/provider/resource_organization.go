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
	_ resource.Resource                = &OrganizationResource{}
	_ resource.ResourceWithImportState = &OrganizationResource{}
	_ resource.ResourceWithConfigure   = &OrganizationResource{}
)

// NewOrganizationResource returns a new organization resource.
func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

// OrganizationResource manages a UbiOps organization.
type OrganizationResource struct {
	client *client.UbiOpsClient
}

// OrganizationResourceModel maps the organization schema to Go types.
type OrganizationResourceModel struct {
	ID                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	Subscription                  types.String `tfsdk:"subscription"`
	SubscriptionEndDate           types.String `tfsdk:"subscription_end_date"`
	SubscriptionStartDate         types.String `tfsdk:"subscription_start_date"`
	TwoFactorAuthenticationForced types.Bool   `tfsdk:"two_factor_authentication_forced"`
	Status                        types.String `tfsdk:"status"`
	CreationDate                  types.String `tfsdk:"creation_date"`
}

func (r *OrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps organization.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the organization (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the organization",
				Required:            true,
			},
			"subscription": schema.StringAttribute{
				MarkdownDescription: "Name of the subscription for the organization",
				Optional:            true,
				Computed:            true,
			},
			"subscription_end_date": schema.StringAttribute{
				MarkdownDescription: "End date of the subscription. The subscription will be cancelled on this date.",
				Optional:            true,
			},
			"subscription_start_date": schema.StringAttribute{
				MarkdownDescription: "Start date of the new subscription. The required format is `YYYY-MM-DD`.",
				Optional:            true,
			},
			"two_factor_authentication_forced": schema.BoolAttribute{
				MarkdownDescription: "Whether 2FA is enforced on the organization users",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the organization",
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "Date and time the organization was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *OrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": data.Name.ValueString(),
	}

	setOptionalString(body, "subscription", data.Subscription)
	setOptionalString(body, "subscription_end_date", data.SubscriptionEndDate)
	setOptionalBool(body, "two_factor_authentication_forced", data.TwoFactorAuthenticationForced)

	// Voucher is a write-only field that can be passed during creation but is not stored in state.

	var result map[string]any
	err := r.client.Post(ctx, "/organizations", body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization", err.Error())
		return
	}

	readOrganizationResult(result, &data)

	tflog.Trace(ctx, "created organization", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrganizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/organizations/%s", data.Name.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	readOrganizationResult(result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationResourceModel
	var state OrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "name", plan.Name, state.Name)
	setChangedString(body, "subscription", plan.Subscription, state.Subscription)
	setChangedBool(body, "two_factor_authentication_forced", plan.TwoFactorAuthenticationForced, state.TwoFactorAuthenticationForced)

	if !plan.SubscriptionEndDate.Equal(state.SubscriptionEndDate) {
		if plan.SubscriptionEndDate.IsNull() {
			body["subscription_end_date"] = nil
		} else {
			body["subscription_end_date"] = plan.SubscriptionEndDate.ValueString()
		}
	}

	if !plan.SubscriptionStartDate.Equal(state.SubscriptionStartDate) {
		if plan.SubscriptionStartDate.IsNull() {
			body["subscription_start_date"] = nil
		} else {
			body["subscription_start_date"] = plan.SubscriptionStartDate.ValueString()
		}
	}

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/organizations/%s", state.Name.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization", err.Error())
		return
	}

	readOrganizationResult(result, &plan)

	tflog.Trace(ctx, "updated organization", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is not supported by the API; removing the resource from state.
func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OrganizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning(
		"Organization not deleted",
		"The UbiOps API does not support deleting organizations. The resource has been removed from Terraform state but the organization still exists.",
	)

	tflog.Warn(ctx, "organization removed from state but not deleted from API", map[string]any{"name": data.Name.ValueString()})
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: organization_name.
	parseOrgImportID(ctx, req.ID, &resp.Diagnostics, resp)
}

// parseOrgImportID parses a single-part import ID for organization resources.
func parseOrgImportID(ctx context.Context, id string, diags *diag.Diagnostics, resp *resource.ImportStateResponse) {
	id = strings.TrimSpace(id)
	if id == "" {
		diags.AddError(
			"Invalid Import ID",
			"Expected format: organization_name",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), id)...)
}

// readOrganizationResult maps the API response to the Terraform model.
func readOrganizationResult(result map[string]any, data *OrganizationResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["status"].(string); ok {
		data.Status = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["subscription"].(string); ok {
		data.Subscription = types.StringValue(v)
	}

	readBoolField(result, "two_factor_authentication_forced", &data.TwoFactorAuthenticationForced)
}
