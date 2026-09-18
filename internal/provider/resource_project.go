// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"

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
	_ resource.Resource                = &ProjectResource{}
	_ resource.ResourceWithImportState = &ProjectResource{}
	_ resource.ResourceWithConfigure   = &ProjectResource{}
)

// NewProjectResource returns a new project resource.
func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource manages a UbiOps project.
type ProjectResource struct {
	client *client.UbiOpsClient
}

// ProjectResourceModel maps the project schema to Go types.
type ProjectResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	OrganizationName    types.String `tfsdk:"organization_name"`
	AdvancedPermissions types.Bool   `tfsdk:"advanced_permissions"`
	Credits             types.Number `tfsdk:"credits"`
	CORSOrigins         types.List   `tfsdk:"cors_origins"`
	Labels              types.Map    `tfsdk:"labels"`
	CreationDate        types.String `tfsdk:"creation_date"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps project.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the project (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project",
				Required:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "Name of the organization in which the project is created",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"advanced_permissions": schema.BoolAttribute{
				MarkdownDescription: "A boolean to enable/disable advanced permissions for the project",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"credits": schema.NumberAttribute{
				MarkdownDescription: "Maximum usage of credits, calculated by multiplying the credit rate of a deployment instance type by the number of hours they are running",
				Optional:            true,
			},
			"cors_origins": schema.ListAttribute{
				MarkdownDescription: "List of origins from which the requests are allowed for the project",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "Time the project was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":              data.Name.ValueString(),
		"organization_name": data.OrganizationName.ValueString(),
	}

	if !data.AdvancedPermissions.IsNull() && !data.AdvancedPermissions.IsUnknown() {
		body["advanced_permissions"] = data.AdvancedPermissions.ValueBool()
	}

	if !data.Credits.IsNull() && !data.Credits.IsUnknown() {
		v, _ := data.Credits.ValueBigFloat().Float64()
		body["credits"] = v
	}

	if !data.CORSOrigins.IsNull() {
		var origins []string
		resp.Diagnostics.Append(data.CORSOrigins.ElementsAs(ctx, &origins, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["cors_origins"] = origins
	}

	if !data.Labels.IsNull() {
		labels := make(map[string]string)
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	var result map[string]any
	err := r.client.Post(ctx, "/projects", body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating project", err.Error())
		return
	}

	readProjectResult(ctx, result, &data)

	tflog.Trace(ctx, "created project", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s", url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project", err.Error())
		return
	}

	readProjectResult(ctx, result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectResourceModel
	var state ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}

	if !plan.AdvancedPermissions.Equal(state.AdvancedPermissions) {
		body["advanced_permissions"] = plan.AdvancedPermissions.ValueBool()
	}

	if !plan.Credits.Equal(state.Credits) {
		if plan.Credits.IsNull() {
			body["credits"] = nil
		} else {
			v, _ := plan.Credits.ValueBigFloat().Float64()
			body["credits"] = v
		}
	}

	if !plan.CORSOrigins.Equal(state.CORSOrigins) {
		if plan.CORSOrigins.IsNull() {
			body["cors_origins"] = []string{}
		} else {
			var origins []string
			resp.Diagnostics.Append(plan.CORSOrigins.ElementsAs(ctx, &origins, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["cors_origins"] = origins
		}
	}

	if !plan.Labels.Equal(state.Labels) {
		if plan.Labels.IsNull() {
			body["labels"] = map[string]string{}
		} else {
			labels := make(map[string]string)
			resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			body["labels"] = labels
		}
	}

	// Use the current name from state for the API path (name may be changing).
	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s", url.PathEscape(state.Name.ValueString())), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating project", err.Error())
		return
	}

	readProjectResult(ctx, result, &plan)

	tflog.Trace(ctx, "updated project", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s", url.PathEscape(data.Name.ValueString())))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting project", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted project", map[string]any{"name": data.Name.ValueString()})
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// readProjectResult maps the API response to the Terraform model.
func readProjectResult(_ context.Context, result map[string]any, data *ProjectResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["organization_name"].(string); ok {
		data.OrganizationName = types.StringValue(v)
	}
	if v, ok := result["advanced_permissions"].(bool); ok {
		data.AdvancedPermissions = types.BoolValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}

	// Credits can be null.
	if v, ok := result["credits"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			data.Credits = types.NumberValue((&types.Number{}).ValueBigFloat().SetFloat64(f))
		}
	} else {
		data.Credits = types.NumberNull()
	}

	// CORS origins: treat empty list same as absent (null) so Optional-only fields stay consistent.
	if v, ok := result["cors_origins"]; ok && v != nil {
		if origins, ok := v.([]any); ok && len(origins) > 0 {
			vals := make([]types.String, 0, len(origins))
			for _, o := range origins {
				if s, ok := o.(string); ok {
					vals = append(vals, types.StringValue(s))
				}
			}
			list, _ := types.ListValueFrom(context.Background(), types.StringType, vals)
			data.CORSOrigins = list
		} else {
			data.CORSOrigins = types.ListNull(types.StringType)
		}
	} else {
		data.CORSOrigins = types.ListNull(types.StringType)
	}

	// Labels: treat empty map same as absent (null) so Optional-only fields stay consistent.
	if v, ok := result["labels"]; ok && v != nil {
		if labelsMap, ok := v.(map[string]any); ok && len(labelsMap) > 0 {
			vals := make(map[string]string, len(labelsMap))
			for k, val := range labelsMap {
				if s, ok := val.(string); ok {
					vals[k] = s
				}
			}
			m, _ := types.MapValueFrom(context.Background(), types.StringType, vals)
			data.Labels = m
		} else {
			data.Labels = types.MapNull(types.StringType)
		}
	} else {
		data.Labels = types.MapNull(types.StringType)
	}
}
