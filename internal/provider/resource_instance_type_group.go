// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
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
	_ resource.Resource                = &InstanceTypeGroupResource{}
	_ resource.ResourceWithImportState = &InstanceTypeGroupResource{}
	_ resource.ResourceWithConfigure   = &InstanceTypeGroupResource{}
)

// NewInstanceTypeGroupResource returns a new instance type group resource.
func NewInstanceTypeGroupResource() resource.Resource {
	return &InstanceTypeGroupResource{}
}

// InstanceTypeGroupResource manages a UbiOps instance type group.
type InstanceTypeGroupResource struct {
	client *client.UbiOpsClient
}

// InstanceTypeGroupResourceModel maps the instance type group schema to Go types.
type InstanceTypeGroupResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectName       types.String `tfsdk:"project_name"`
	Name              types.String `tfsdk:"name"`
	InstanceTypesJSON types.String `tfsdk:"instance_types_json"`
	TimeCreated       types.String `tfsdk:"time_created"`
	TimeUpdated       types.String `tfsdk:"time_updated"`
}

func (r *InstanceTypeGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_type_group"
}

func (r *InstanceTypeGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps instance type group.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the created instance type group (UUID)",
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
				MarkdownDescription: "Name of the instance type group",
				Required:            true,
			},
			"instance_types_json": schema.StringAttribute{
				MarkdownDescription: "A list of instance types that are in this group.",
				Required:            true,
			},
			"time_created": schema.StringAttribute{
				MarkdownDescription: "The date when the instance type group was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"time_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the instance type group was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *InstanceTypeGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *InstanceTypeGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InstanceTypeGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": data.Name.ValueString(),
	}

	if !data.InstanceTypesJSON.IsNull() && !data.InstanceTypesJSON.IsUnknown() {
		var instanceTypes []any
		if err := json.Unmarshal([]byte(data.InstanceTypesJSON.ValueString()), &instanceTypes); err != nil {
			resp.Diagnostics.AddError("Invalid instance_types_json", err.Error())
			return
		}
		body["instance_types"] = instanceTypes
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/instance-type-groups", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating instance type group", err.Error())
		return
	}

	readInstanceTypeGroupResult(result, &data)
	data.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "created instance type group", map[string]any{"id": data.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceTypeGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InstanceTypeGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/instance-type-groups/%s", projectName, data.ID.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading instance type group", err.Error())
		return
	}

	readInstanceTypeGroupResult(result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InstanceTypeGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceTypeGroupResourceModel
	var state InstanceTypeGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}

	if !plan.InstanceTypesJSON.Equal(state.InstanceTypesJSON) {
		if plan.InstanceTypesJSON.IsNull() {
			body["instance_types"] = []any{}
		} else {
			var instanceTypes []any
			if err := json.Unmarshal([]byte(plan.InstanceTypesJSON.ValueString()), &instanceTypes); err != nil {
				resp.Diagnostics.AddError("Invalid instance_types_json", err.Error())
				return
			}
			body["instance_types"] = instanceTypes
		}
	}

	projectName := state.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/instance-type-groups/%s", projectName, state.ID.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating instance type group", err.Error())
		return
	}

	readInstanceTypeGroupResult(result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated instance type group", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InstanceTypeGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data InstanceTypeGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/instance-type-groups/%s", data.ProjectName.ValueString(), data.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting instance type group", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted instance type group", map[string]any{"id": data.ID.ValueString()})
}

func (r *InstanceTypeGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "id", resp)
}

// readInstanceTypeGroupResult maps the API response to the Terraform model.
func readInstanceTypeGroupResult(result map[string]any, data *InstanceTypeGroupResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["time_created"].(string); ok {
		data.TimeCreated = types.StringValue(v)
	}
	if v, ok := result["time_updated"].(string); ok {
		data.TimeUpdated = types.StringValue(v)
	}

	// Instance types — strip API-added fields, store only the user-controlled keys.
	// The API enriches each entry with cpu, memory, credit_rate, display_name, etc.
	// Storing the full blob causes state drift: config has 3-key objects but state
	// has 10+ key objects. Only keep id, priority - schedule_timeout is a pure
	// server-computed, output-only field (not settable via the API, and its value
	// varies) so it's never tracked in state, matching what the user actually configures.
	if v, ok := result["instance_types"]; ok && v != nil {
		if arr, ok := v.([]any); ok {
			stripped := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					entry := map[string]any{}
					if id, ok := m["id"]; ok {
						entry["id"] = id
					}
					if p, ok := m["priority"]; ok {
						entry["priority"] = p
					}
					stripped = append(stripped, entry)
				}
			}
			b, err := json.Marshal(stripped)
			if err == nil {
				data.InstanceTypesJSON = types.StringValue(string(b))
			}
		}
	} else {
		data.InstanceTypesJSON = types.StringNull()
	}
}
