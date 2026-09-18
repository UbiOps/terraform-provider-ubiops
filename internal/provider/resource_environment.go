// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &EnvironmentResource{}
	_ resource.ResourceWithImportState = &EnvironmentResource{}
	_ resource.ResourceWithConfigure   = &EnvironmentResource{}
)

// NewEnvironmentResource returns a new environment resource.
func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

// EnvironmentResource manages a UbiOps environment.
type EnvironmentResource struct {
	client *client.UbiOpsClient
}

// EnvironmentResourceModel maps the environment schema to Go types.
type EnvironmentResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ProjectName           types.String `tfsdk:"project_name"`
	Name                  types.String `tfsdk:"name"`
	DisplayName           types.String `tfsdk:"display_name"`
	BaseEnvironment       types.String `tfsdk:"base_environment"`
	Description           types.String `tfsdk:"description"`
	SupportsRequestFormat types.Bool   `tfsdk:"supports_request_format"`
	Labels                types.Map    `tfsdk:"labels"`
	SourceFile            types.String `tfsdk:"source_file"`
	SourceFileSHA256      types.String `tfsdk:"source_file_sha256"`
	BuildTimeout          types.Int64  `tfsdk:"build_timeout"`
	CreationDate          types.String `tfsdk:"creation_date"`
	LastUpdated           types.String `tfsdk:"last_updated"`
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps environment.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the environment",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project this environment belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the environment",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"base_environment": schema.StringAttribute{
				MarkdownDescription: "Base environment name on which this environment is based",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the environment",
				Optional:            true,
				Computed:            true,
			},
			"supports_request_format": schema.BoolAttribute{
				MarkdownDescription: "Whether the environment supports UbiOps's structured request format (queuing, autoscaling, scheduled requests). Must match the `supports_request_format` of any deployment using this environment, or version creation fails.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"source_file": schema.StringAttribute{
				MarkdownDescription: "Path to a requirements.txt or zip file to upload as a revision. Triggers an environment build.",
				Optional:            true,
			},
			"source_file_sha256": schema.StringAttribute{
				MarkdownDescription: "SHA-256 of the source file. Set to `filesha256(\"path\")` to detect content changes. Computed automatically if not set.",
				Optional:            true,
				Computed:            true,
			},
			"build_timeout": schema.Int64Attribute{
				MarkdownDescription: "Seconds to wait for the environment build to complete after uploading a revision. Defaults to 1800 (30 min). Set to 0 to skip waiting.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1800),
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the environment was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the environment was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *EnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":                    data.Name.ValueString(),
		"supports_request_format": data.SupportsRequestFormat.ValueBool(),
	}

	if !data.DisplayName.IsNull() && !data.DisplayName.IsUnknown() {
		body["display_name"] = data.DisplayName.ValueString()
	}

	if !data.BaseEnvironment.IsNull() && !data.BaseEnvironment.IsUnknown() {
		body["base_environment"] = data.BaseEnvironment.ValueString()
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}

	if !data.Labels.IsNull() {
		labels := make(map[string]string)
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/environments", url.PathEscape(projectName)), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment", err.Error())
		return
	}

	readEnvironmentResult(result, &data)
	data.ProjectName = types.StringValue(projectName)

	// Save state before upload so the resource is tracked even if upload fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.SourceFile.IsNull() && !data.SourceFile.IsUnknown() {
		revisionID := r.uploadRevision(ctx, &data, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
		r.waitForBuild(ctx, &data, revisionID, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		data.SourceFileSHA256 = types.StringNull()
	}

	tflog.Trace(ctx, "created environment", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/environments/%s", url.PathEscape(projectName), url.PathEscape(data.Name.ValueString())), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment", err.Error())
		return
	}

	readEnvironmentResult(result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentResourceModel
	var state EnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}

	if !plan.DisplayName.Equal(state.DisplayName) {
		body["display_name"] = plan.DisplayName.ValueString()
	}

	if !plan.Description.Equal(state.Description) {
		body["description"] = plan.Description.ValueString()
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

	projectName := state.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("/projects/%s/environments/%s", url.PathEscape(projectName), url.PathEscape(state.Name.ValueString())), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment", err.Error())
		return
	}

	// Save source_file/sha256 - PATCH response omits source_file_sha256,
	// so re-upload check would see null==null otherwise.
	wantSourceFile := plan.SourceFile
	wantSourceFileSHA256 := plan.SourceFileSHA256

	readEnvironmentResult(result, &plan)
	plan.ProjectName = types.StringValue(projectName)
	plan.SourceFile = wantSourceFile
	plan.SourceFileSHA256 = wantSourceFileSHA256

	if !plan.SourceFile.IsNull() && !plan.SourceFile.IsUnknown() {
		if !plan.SourceFile.Equal(state.SourceFile) || !plan.SourceFileSHA256.Equal(state.SourceFileSHA256) {
			revisionID := r.uploadRevision(ctx, &plan, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
			r.waitForBuild(ctx, &plan, revisionID, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		} else {
			plan.SourceFileSHA256 = state.SourceFileSHA256
		}
	} else {
		plan.SourceFileSHA256 = types.StringNull()
	}

	tflog.Trace(ctx, "updated environment", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/environments/%s", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString())))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting environment", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted environment", map[string]any{"name": data.Name.ValueString()})
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// readEnvironmentResult maps the API response to the Terraform model.
func readEnvironmentResult(result map[string]any, data *EnvironmentResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["display_name"].(string); ok {
		data.DisplayName = types.StringValue(v)
	} else {
		data.DisplayName = types.StringNull()
	}
	if v, ok := result["base_environment"].(string); ok {
		data.BaseEnvironment = types.StringValue(v)
	} else {
		data.BaseEnvironment = types.StringNull()
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	} else {
		data.Description = types.StringNull()
	}
	if v, ok := result["supports_request_format"].(bool); ok {
		data.SupportsRequestFormat = types.BoolValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
	}

	// Labels: store null when empty so plan null stays null.
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

// uploadRevision uploads to the revisions endpoint, returning the new revision's id.
func (r *EnvironmentResource) uploadRevision(ctx context.Context, data *EnvironmentResourceModel, diags *diag.Diagnostics) string {
	filePath := data.SourceFile.ValueString()

	hash, err := computeFileSHA256(filePath)
	if err != nil {
		diags.AddError("Error computing source file hash", err.Error())
		return ""
	}
	if data.SourceFileSHA256.IsNull() || data.SourceFileSHA256.IsUnknown() {
		data.SourceFileSHA256 = types.StringValue(hash)
	}

	revisionPath := fmt.Sprintf("/projects/%s/environments/%s/revisions", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString()))

	var result map[string]any
	if err := r.client.Upload(ctx, revisionPath, filePath, &result); err != nil {
		diags.AddError("Error uploading environment revision", err.Error())
		return ""
	}

	tflog.Info(ctx, "uploaded environment revision", map[string]any{"name": data.Name.ValueString()})
	revisionID, _ := result["revision"].(string)
	return revisionID
}

// waitForBuild polls the uploaded revision's build, not the environment's own
// "status" (which reflects the previous build until the new one finishes).
func (r *EnvironmentResource) waitForBuild(ctx context.Context, data *EnvironmentResourceModel, revisionID string, diags *diag.Diagnostics) {
	timeoutSecs := data.BuildTimeout.ValueInt64()
	if timeoutSecs == 0 || revisionID == "" {
		return
	}

	buildsPath := fmt.Sprintf("/projects/%s/environments/%s/revisions/%s/builds", url.PathEscape(data.ProjectName.ValueString()), url.PathEscape(data.Name.ValueString()), url.PathEscape(revisionID))
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)

	tflog.Info(ctx, "waiting for environment build", map[string]any{
		"name":    data.Name.ValueString(),
		"timeout": timeoutSecs,
	})

	for {
		var builds []map[string]any
		if err := r.client.Get(ctx, buildsPath, &builds); err != nil {
			diags.AddError("Error polling environment build", err.Error())
			return
		}

		var status string
		var errMsg string
		if len(builds) > 0 {
			status, _ = builds[0]["status"].(string)
			errMsg, _ = builds[0]["error_message"].(string)
		}

		switch status {
		case "success":
			tflog.Info(ctx, "environment build complete", map[string]any{"name": data.Name.ValueString()})
			return
		case "failed":
			diags.AddError(
				fmt.Sprintf("Environment %q build failed", data.Name.ValueString()),
				errMsg,
			)
			return
		}

		if time.Now().After(deadline) {
			diags.AddError(
				fmt.Sprintf("Environment %q build timed out", data.Name.ValueString()),
				fmt.Sprintf("still %q after %d seconds; increase build_timeout or set it to 0 to skip", status, timeoutSecs),
			)
			return
		}

		select {
		case <-ctx.Done():
			diags.AddError("Context cancelled while waiting for environment build", ctx.Err().Error())
			return
		case <-time.After(15 * time.Second):
		}
	}
}
