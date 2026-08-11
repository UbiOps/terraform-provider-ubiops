// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &DeploymentVersionResource{}
	_ resource.ResourceWithImportState = &DeploymentVersionResource{}
	_ resource.ResourceWithConfigure   = &DeploymentVersionResource{}
)

// NewDeploymentVersionResource returns a new deployment version resource.
func NewDeploymentVersionResource() resource.Resource {
	return &DeploymentVersionResource{}
}

// DeploymentVersionResource manages a UbiOps deployment version.
type DeploymentVersionResource struct {
	client *client.UbiOpsClient
}

// DeploymentVersionResourceModel maps the deployment version schema to Go types.
type DeploymentVersionResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	ProjectName                types.String `tfsdk:"project_name"`
	DeploymentName             types.String `tfsdk:"deployment_name"`
	Version                    types.String `tfsdk:"version"`
	Description                types.String `tfsdk:"description"`
	Environment                types.String `tfsdk:"environment"`
	InstanceTypeGroupName      types.String `tfsdk:"instance_type_group_name"`
	MinimumInstances           types.Int64  `tfsdk:"minimum_instances"`
	MaximumInstances           types.Int64  `tfsdk:"maximum_instances"`
	MaximumIdleTime            types.Int64  `tfsdk:"maximum_idle_time"`
	MaximumQueueSize           types.Int64  `tfsdk:"maximum_queue_size"`
	InstanceProcesses          types.Int64  `tfsdk:"instance_processes"`
	RequestRetentionMode       types.String `tfsdk:"request_retention_mode"`
	RequestRetentionTime       types.Int64  `tfsdk:"request_retention_time"`
	ScalingStrategy            types.String `tfsdk:"scaling_strategy"`
	StaticIP                   types.Bool   `tfsdk:"static_ip"`
	RestartRequestInterruption types.Bool   `tfsdk:"restart_request_interruption"`
	Labels                     types.Map    `tfsdk:"labels"`
	SourceFile                 types.String `tfsdk:"source_file"`
	SourceFileSHA256           types.String `tfsdk:"source_file_sha256"`
	BuildTimeout               types.Int64  `tfsdk:"build_timeout"`
	CreationDate               types.String `tfsdk:"creation_date"`
	LastUpdated                types.String `tfsdk:"last_updated"`
}

func (r *DeploymentVersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_version"
}

func (r *DeploymentVersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps deployment version.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the deployment version (UUID)",
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
			"deployment_name": schema.StringAttribute{
				MarkdownDescription: "The name of the deployment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the version",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Environment of the version",
				Optional:            true,
				Computed:            true,
			},
			"instance_type_group_name": schema.StringAttribute{
				MarkdownDescription: "Name of the instance type group for the version",
				Optional:            true,
				Computed:            true,
			},
			"minimum_instances": schema.Int64Attribute{
				MarkdownDescription: "Lower bound of number of instances running",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"maximum_instances": schema.Int64Attribute{
				MarkdownDescription: "Upper bound of number of instances running",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(5),
			},
			"maximum_idle_time": schema.Int64Attribute{
				MarkdownDescription: "Maximum time in seconds a version stays idle before it is stopped",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(300),
			},
			"maximum_queue_size": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of queued requests for all instances of this deployment version",
				Optional:            true,
				Computed:            true,
			},
			"instance_processes": schema.Int64Attribute{
				MarkdownDescription: "Number of processes that are started in each instance",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
			},
			"request_retention_mode": schema.StringAttribute{
				MarkdownDescription: "Mode of request retention for requests to the version. It can be one of the following: - *none* - the requests will not be stored - *metadata* - only the metadata of the requests will be stored - *full* - both the metadata and input/output of the requests will be stored.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("full"),
				Validators: []validator.String{
					stringvalidator.OneOf("none", "metadata", "full"),
				},
			},
			"request_retention_time": schema.Int64Attribute{
				MarkdownDescription: "Number of seconds to store requests to the version",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(604800),
			},
			"scaling_strategy": schema.StringAttribute{
				MarkdownDescription: "Scaling strategy for running instances. It can be one of the following: *default* or *moderate*.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				Validators: []validator.String{
					stringvalidator.OneOf("default", "moderate"),
				},
			},
			"static_ip": schema.BoolAttribute{
				MarkdownDescription: "A boolean indicating whether the deployment version should get a static IP. The UbiOps API does not return this value on read, so it is not verified when importing an existing version.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"restart_request_interruption": schema.BoolAttribute{
				MarkdownDescription: "A boolean indicating whether the requests should be restarted in case of an interruption",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"source_file": schema.StringAttribute{
				MarkdownDescription: "Path to a zip file containing the deployment package to upload as a revision. Changing this value or `source_file_sha256` triggers a new revision upload.",
				Optional:            true,
			},
			"source_file_sha256": schema.StringAttribute{
				MarkdownDescription: "SHA-256 hash of the source file. Used to detect changes to the deployment package. If not set, a hash is computed automatically from `source_file`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"build_timeout": schema.Int64Attribute{
				MarkdownDescription: "Seconds to wait for the deployment package build to complete after uploading. Defaults to 1800 (30 min). Set to 0 to skip waiting (restores the previous two-pass apply behaviour).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1800),
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the version was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the version was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *DeploymentVersionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *DeploymentVersionResource) basePath(projectName, deploymentName string) string {
	return fmt.Sprintf("/projects/%s/deployments/%s/versions", projectName, deploymentName)
}

func (r *DeploymentVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentVersionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"version": data.Version.ValueString(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalString(body, "environment", data.Environment)
	setOptionalString(body, "instance_type_group_name", data.InstanceTypeGroupName)
	setOptionalString(body, "request_retention_mode", data.RequestRetentionMode)
	setOptionalString(body, "scaling_strategy", data.ScalingStrategy)
	setOptionalInt64(body, "minimum_instances", data.MinimumInstances)
	setOptionalInt64(body, "maximum_instances", data.MaximumInstances)
	setOptionalInt64(body, "maximum_idle_time", data.MaximumIdleTime)
	setOptionalInt64(body, "maximum_queue_size", data.MaximumQueueSize)
	setOptionalInt64(body, "instance_processes", data.InstanceProcesses)
	setOptionalInt64(body, "request_retention_time", data.RequestRetentionTime)
	setOptionalBool(body, "static_ip", data.StaticIP)
	setOptionalBool(body, "restart_request_interruption", data.RestartRequestInterruption)

	if !data.Labels.IsNull() {
		labels := make(map[string]string)
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString())

	var result map[string]any
	err := r.client.Post(ctx, bp, body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating deployment version", err.Error())
		return
	}

	readDeploymentVersionResult(ctx, result, &data)

	// The API's version-create response never includes a source_file_sha256 field (it's a
	// Terraform-only computed attribute for tracking uploaded revisions, not something the
	// API tracks). When source_file isn't set at all (referencing an existing environment,
	// no upload happening), the upload branch below - the only other place that resolves
	// this field - never runs either, leaving it Unknown when saved to state, which
	// Terraform rejects ("provider produced inconsistent result"). Resolve it to a known
	// null explicitly in that case.
	if data.SourceFile.IsNull() || data.SourceFile.IsUnknown() {
		data.SourceFileSHA256 = types.StringNull()
	}

	// Save state immediately so Terraform tracks the resource even if the upload fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Upload deployment package if source_file is set.
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
	}

	tflog.Trace(ctx, "created deployment version", map[string]any{"version": data.Version.ValueString()})
}

func (r *DeploymentVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString())

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("%s/%s", bp, data.Version.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading deployment version", err.Error())
		return
	}

	readDeploymentVersionResult(ctx, result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentVersionResourceModel
	var state DeploymentVersionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "version", plan.Version, state.Version)
	setChangedString(body, "description", plan.Description, state.Description)
	setChangedString(body, "environment", plan.Environment, state.Environment)
	setChangedString(body, "instance_type_group_name", plan.InstanceTypeGroupName, state.InstanceTypeGroupName)
	setChangedString(body, "request_retention_mode", plan.RequestRetentionMode, state.RequestRetentionMode)
	setChangedString(body, "scaling_strategy", plan.ScalingStrategy, state.ScalingStrategy)
	setChangedInt64(body, "minimum_instances", plan.MinimumInstances, state.MinimumInstances)
	setChangedInt64(body, "maximum_instances", plan.MaximumInstances, state.MaximumInstances)
	setChangedInt64(body, "maximum_idle_time", plan.MaximumIdleTime, state.MaximumIdleTime)
	setChangedInt64(body, "maximum_queue_size", plan.MaximumQueueSize, state.MaximumQueueSize)
	setChangedInt64(body, "instance_processes", plan.InstanceProcesses, state.InstanceProcesses)
	setChangedInt64(body, "request_retention_time", plan.RequestRetentionTime, state.RequestRetentionTime)
	setChangedBool(body, "static_ip", plan.StaticIP, state.StaticIP)
	setChangedBool(body, "restart_request_interruption", plan.RestartRequestInterruption, state.RestartRequestInterruption)

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

	bp := r.basePath(state.ProjectName.ValueString(), state.DeploymentName.ValueString())

	var result map[string]any
	err := r.client.Patch(ctx, fmt.Sprintf("%s/%s", bp, state.Version.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating deployment version", err.Error())
		return
	}

	readDeploymentVersionResult(ctx, result, &plan)

	// Upload a new revision if source_file or its hash changed.
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
	}

	tflog.Trace(ctx, "updated deployment version", map[string]any{"version": plan.Version.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DeploymentVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString())

	err := r.client.Delete(ctx, fmt.Sprintf("%s/%s", bp, data.Version.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting deployment version", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted deployment version", map[string]any{"version": data.Version.ValueString()})
}

// uploadRevision uploads the deployment package zip to the revisions endpoint.
// Returns the created revision's ID (empty string if the API response didn't
// include one, e.g. an unexpected response shape) for waitForBuild to poll.
func (r *DeploymentVersionResource) uploadRevision(ctx context.Context, data *DeploymentVersionResourceModel, diags *diag.Diagnostics) string {
	filePath := data.SourceFile.ValueString()

	hash, err := computeFileSHA256(filePath)
	if err != nil {
		diags.AddError("Error computing source file hash", err.Error())
		return ""
	}

	// If the user provided a hash, use it; otherwise use the computed one.
	if data.SourceFileSHA256.IsNull() || data.SourceFileSHA256.IsUnknown() {
		data.SourceFileSHA256 = types.StringValue(hash)
	}

	revisionPath := fmt.Sprintf(
		"/projects/%s/deployments/%s/versions/%s/revisions",
		data.ProjectName.ValueString(),
		data.DeploymentName.ValueString(),
		data.Version.ValueString(),
	)

	var result map[string]any
	err = r.client.Upload(ctx, revisionPath, filePath, &result)
	if err != nil {
		diags.AddError("Error uploading deployment package", err.Error())
		return ""
	}

	revisionID, _ := result["revision"].(string)

	tflog.Info(ctx, "uploaded deployment package", map[string]any{
		"version":     data.Version.ValueString(),
		"source_file": filePath,
		"revision_id": revisionID,
	})
	return revisionID
}

// waitForBuild polls the REVISION's own build status (success/failed) until
// it completes or the timeout elapses. Deliberately does NOT poll the
// version's aggregate "status" field (available/unavailable) - that field
// tracks whether a serving instance is up, which never happens while
// minimum_instances == 0. Polling it in that case blocks for the full
// timeout even though the package build itself finished in seconds
// (confirmed live 2026-07-16: build succeeded per the revision's own status,
// but version stayed "unavailable" indefinitely with minimum_instances=0).
// Falls back to the old version-status polling if no revisionID is
// available (e.g. an unexpected upload response shape), so an empty
// revisionID degrades rather than silently skipping the wait.
// Set build_timeout = 0 to skip polling entirely.
func (r *DeploymentVersionResource) waitForBuild(ctx context.Context, data *DeploymentVersionResourceModel, revisionID string, diags *diag.Diagnostics) {
	timeoutSecs := data.BuildTimeout.ValueInt64()
	if timeoutSecs == 0 {
		return
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.DeploymentName.ValueString())
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	var pollPath string
	if revisionID != "" {
		pollPath = fmt.Sprintf("%s/%s/revisions/%s", bp, data.Version.ValueString(), revisionID)
	} else {
		pollPath = fmt.Sprintf("%s/%s", bp, data.Version.ValueString())
	}

	tflog.Info(ctx, "waiting for deployment build", map[string]any{
		"version":     data.Version.ValueString(),
		"revision_id": revisionID,
		"timeout":     timeoutSecs,
	})

	for {
		var result map[string]any
		if err := r.client.Get(ctx, pollPath, &result); err != nil {
			diags.AddError("Error polling deployment version build", err.Error())
			return
		}

		status, _ := result["status"].(string)
		successStatus := "available"
		if revisionID != "" {
			successStatus = "success"
		}
		switch status {
		case successStatus:
			tflog.Info(ctx, "deployment build complete", map[string]any{"version": data.Version.ValueString()})
			return
		case "failed":
			errMsg, _ := result["error_message"].(string)
			diags.AddError(
				fmt.Sprintf("Deployment version %q build failed", data.Version.ValueString()),
				errMsg,
			)
			return
		}

		if time.Now().After(deadline) {
			diags.AddError(
				fmt.Sprintf("Deployment version %q build timed out", data.Version.ValueString()),
				fmt.Sprintf("still %q after %d seconds; increase build_timeout or set it to 0 to skip waiting", status, timeoutSecs),
			)
			return
		}

		select {
		case <-ctx.Done():
			diags.AddError("Context cancelled while waiting for build", ctx.Err().Error())
			return
		case <-time.After(15 * time.Second):
		}
	}
}

// computeFileSHA256 returns the hex-encoded SHA-256 hash of a file.
func computeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file %s: %w", path, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (r *DeploymentVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID3(ctx, req.ID, "deployment_name", "version", resp)
}

// readDeploymentVersionResult maps the API response to the Terraform model.
func readDeploymentVersionResult(ctx context.Context, result map[string]any, data *DeploymentVersionResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["version"].(string); ok {
		data.Version = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["environment"].(string); ok {
		data.Environment = types.StringValue(v)
	}
	if v, ok := result["instance_type_group_name"].(string); ok {
		data.InstanceTypeGroupName = types.StringValue(v)
	}
	if v, ok := result["request_retention_mode"].(string); ok {
		data.RequestRetentionMode = types.StringValue(v)
	}
	if v, ok := result["scaling_strategy"].(string); ok {
		data.ScalingStrategy = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
	}

	readInt64Field(result, "minimum_instances", &data.MinimumInstances)
	readInt64Field(result, "maximum_instances", &data.MaximumInstances)
	readInt64Field(result, "maximum_idle_time", &data.MaximumIdleTime)
	readInt64Field(result, "maximum_queue_size", &data.MaximumQueueSize)
	readInt64Field(result, "instance_processes", &data.InstanceProcesses)
	readInt64Field(result, "request_retention_time", &data.RequestRetentionTime)

	readBoolField(result, "static_ip", &data.StaticIP)
	readBoolField(result, "restart_request_interruption", &data.RestartRequestInterruption)

	// Labels.
	// Labels: treat empty map same as absent so Optional-only field stays consistent.
	if v, ok := result["labels"]; ok && v != nil {
		if labelsMap, ok := v.(map[string]any); ok && len(labelsMap) > 0 {
			vals := make(map[string]string, len(labelsMap))
			for k, val := range labelsMap {
				if s, ok := val.(string); ok {
					vals[k] = s
				}
			}
			m, _ := types.MapValueFrom(ctx, types.StringType, vals)
			data.Labels = m
		} else {
			data.Labels = types.MapNull(types.StringType)
		}
	} else {
		data.Labels = types.MapNull(types.StringType)
	}
}

// Helper functions for building request bodies and reading responses.

func setOptionalString(body map[string]any, key string, val types.String) {
	if !val.IsNull() && !val.IsUnknown() {
		body[key] = val.ValueString()
	}
}

func setOptionalInt64(body map[string]any, key string, val types.Int64) {
	if !val.IsNull() && !val.IsUnknown() {
		body[key] = val.ValueInt64()
	}
}

func setOptionalBool(body map[string]any, key string, val types.Bool) {
	if !val.IsNull() && !val.IsUnknown() {
		body[key] = val.ValueBool()
	}
}

func setChangedString(body map[string]any, key string, plan, state types.String) {
	if !plan.Equal(state) && !plan.IsNull() && !plan.IsUnknown() {
		body[key] = plan.ValueString()
	}
}

func setChangedInt64(body map[string]any, key string, plan, state types.Int64) {
	if !plan.Equal(state) && !plan.IsNull() && !plan.IsUnknown() {
		body[key] = plan.ValueInt64()
	}
}

func setChangedBool(body map[string]any, key string, plan, state types.Bool) {
	if !plan.Equal(state) && !plan.IsNull() && !plan.IsUnknown() {
		body[key] = plan.ValueBool()
	}
}

func readInt64Field(result map[string]any, key string, target *types.Int64) {
	if v, ok := result[key]; ok && v != nil {
		if f, ok := v.(float64); ok {
			*target = types.Int64Value(int64(f))
		}
	}
}

func readBoolField(result map[string]any, key string, target *types.Bool) {
	if v, ok := result[key]; ok && v != nil {
		if b, ok := v.(bool); ok {
			*target = types.BoolValue(b)
		}
	}
}
