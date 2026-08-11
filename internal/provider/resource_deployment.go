// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &DeploymentResource{}
	_ resource.ResourceWithImportState = &DeploymentResource{}
	_ resource.ResourceWithConfigure   = &DeploymentResource{}
)

// NewDeploymentResource returns a new deployment resource.
func NewDeploymentResource() resource.Resource {
	return &DeploymentResource{}
}

// DeploymentResource manages a UbiOps deployment.
type DeploymentResource struct {
	client *client.UbiOpsClient
}

// DeploymentResourceModel maps the deployment schema to Go types.
type DeploymentResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	ProjectName           types.String `tfsdk:"project_name"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	InputType             types.String `tfsdk:"input_type"`
	OutputType            types.String `tfsdk:"output_type"`
	SupportsRequestFormat types.Bool   `tfsdk:"supports_request_format"`
	InputFields           types.List   `tfsdk:"input_fields"`
	OutputFields          types.List   `tfsdk:"output_fields"`
	Labels                types.Map    `tfsdk:"labels"`
	DefaultVersion        types.String `tfsdk:"default_version"`
	CreationDate          types.String `tfsdk:"creation_date"`
	LastUpdated           types.String `tfsdk:"last_updated"`
}

// fieldObjectType defines the object type for input/output fields.
var fieldObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"data_type": types.StringType,
	},
}

func (r *DeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *DeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps deployment.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the deployment (UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project this deployment belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the deployment",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the deployment",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"input_type": schema.StringAttribute{
				MarkdownDescription: "Type of the input of the deployment",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("structured"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("structured", "plain"),
				},
			},
			"output_type": schema.StringAttribute{
				MarkdownDescription: "Type of the output of the deployment",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("structured"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("structured", "plain"),
				},
			},
			"supports_request_format": schema.BoolAttribute{
				MarkdownDescription: "Whether the deployment supports UbiOps's structured request format (queuing, autoscaling, scheduled requests). false is required for Bring Your Own Docker deployments that are standalone applications (e.g. a Jupyter server, or a raw HTTP service like litellm-gateway) - the deployment's environment must have a matching supports_request_format value, or version creation fails with \"Given environment is not supported for the deployment\". Deployments with this set to false have no autoscaling - a fixed instance count only.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"input_fields": schema.ListNestedAttribute{
				MarkdownDescription: "The list of deployment input fields containing name and data_type. It is empty in case of plain input type deployments.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the field.",
							Required:            true,
						},
						"data_type": schema.StringAttribute{
							MarkdownDescription: "The data type of the field.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"int", "string", "double", "bool", "dict",
									"array_int", "array_double", "array_string",
									"file", "array_file",
								),
							},
						},
					},
				},
			},
			"output_fields": schema.ListNestedAttribute{
				MarkdownDescription: "The list of deployment output fields containing name and data_type. It is empty in case of plain output type deployments.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the field.",
							Required:            true,
						},
						"data_type": schema.StringAttribute{
							MarkdownDescription: "The data type of the field.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"int", "string", "double", "bool", "dict",
									"array_int", "array_double", "array_string",
									"file", "array_file",
								),
							},
						},
					},
				},
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Dictionary containing key/value pairs where key indicates the label and value is the corresponding value of that label",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"default_version": schema.StringAttribute{
				MarkdownDescription: "Default version of the deployment. If it does not have a default version, it is not set. Set this to promote a version to receive traffic without recreating the deployment — the standard blue/green pattern for zero-downtime version rollouts and rollbacks. Setting it on create (rather than a later update) works too: Create() waits for the first ubiops_deployment_version under this deployment to auto-promote to that value - see waitForDefaultVersion.",
				Optional:            true,
				Computed:            true,
			},
			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the deployment was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the deployment was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *DeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *DeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name":                    data.Name.ValueString(),
		"input_type":              data.InputType.ValueString(),
		"output_type":             data.OutputType.ValueString(),
		"supports_request_format": data.SupportsRequestFormat.ValueBool(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		body["description"] = data.Description.ValueString()
	}

	if !data.InputFields.IsNull() && !data.InputFields.IsUnknown() {
		body["input_fields"] = fieldsToAPI(ctx, data.InputFields, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.OutputFields.IsNull() && !data.OutputFields.IsUnknown() {
		body["output_fields"] = fieldsToAPI(ctx, data.OutputFields, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
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
	err := r.client.Post(ctx, fmt.Sprintf("/projects/%s/deployments", projectName), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error creating deployment", err.Error())
		return
	}

	// Capture default_version before readDeploymentResult resets it to null - the create
	// response never includes it (see waitForDefaultVersion).
	wantDefaultVersion := data.DefaultVersion

	readDeploymentResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)

	if !wantDefaultVersion.IsNull() && !wantDefaultVersion.IsUnknown() {
		r.waitForDefaultVersion(ctx, &data, wantDefaultVersion.ValueString(), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Trace(ctx, "created deployment", map[string]any{"name": data.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// timeout/interval for waitForDefaultVersion's polling.
const (
	defaultVersionCreateWaitTimeout  = 5 * time.Minute
	defaultVersionCreateWaitInterval = 2 * time.Second
)

// waitForDefaultVersion polls until default_version becomes want (create never sets it).
func (r *DeploymentResource) waitForDefaultVersion(ctx context.Context, data *DeploymentResourceModel, want string, diags *diag.Diagnostics) {
	path := fmt.Sprintf("/projects/%s/deployments/%s", data.ProjectName.ValueString(), data.Name.ValueString())
	deadline := time.Now().Add(defaultVersionCreateWaitTimeout)

	for {
		var result map[string]any
		if err := r.client.Get(ctx, path, &result); err != nil {
			diags.AddError("Error polling deployment for default_version", err.Error())
			return
		}

		if dv, _ := result["default_version"].(string); dv == want {
			readDeploymentResult(ctx, result, data)
			return
		}

		if time.Now().After(deadline) {
			diags.AddError(
				fmt.Sprintf("Deployment %q did not reach default_version %q", data.Name.ValueString(), want),
				"a ubiops_deployment_version targeting this deployment auto-promotes to default as soon as it's created - check that one exists in the same apply and reached the API within the timeout",
			)
			return
		}

		tflog.Info(ctx, "waiting for a version to become default", map[string]any{"want": want})
		select {
		case <-time.After(defaultVersionCreateWaitInterval):
		case <-ctx.Done():
			diags.AddError("Context cancelled while waiting for default_version", ctx.Err().Error())
			return
		}
	}
}

func (r *DeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectName := data.ProjectName.ValueString()

	var result map[string]any
	err := r.client.Get(ctx, fmt.Sprintf("/projects/%s/deployments/%s", projectName, data.Name.ValueString()), &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading deployment", err.Error())
		return
	}

	readDeploymentResult(ctx, result, &data)
	data.ProjectName = types.StringValue(projectName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentResourceModel
	var state DeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}

	if !plan.Description.Equal(state.Description) {
		body["description"] = plan.Description.ValueString()
	}

	if !plan.InputFields.Equal(state.InputFields) {
		if plan.InputFields.IsNull() {
			body["input_fields"] = []any{}
		} else {
			body["input_fields"] = fieldsToAPI(ctx, plan.InputFields, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	if !plan.OutputFields.Equal(state.OutputFields) {
		if plan.OutputFields.IsNull() {
			body["output_fields"] = []any{}
		} else {
			body["output_fields"] = fieldsToAPI(ctx, plan.OutputFields, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
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

	setChangedString(body, "default_version", plan.DefaultVersion, state.DefaultVersion)

	projectName := state.ProjectName.ValueString()

	var result map[string]any
	err := r.patchDeployment(ctx, fmt.Sprintf("/projects/%s/deployments/%s", projectName, state.Name.ValueString()), body, &result)
	if err != nil {
		resp.Diagnostics.AddError("Error updating deployment", err.Error())
		return
	}

	readDeploymentResult(ctx, result, &plan)
	plan.ProjectName = types.StringValue(projectName)

	tflog.Trace(ctx, "updated deployment", map[string]any{"name": plan.Name.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// timeout/interval for patchDeployment's polling.
const (
	defaultVersionUnavailableRetryTimeout  = 30 * time.Minute
	defaultVersionUnavailableRetryInterval = 5 * time.Second
)

// patchDeployment retries default_version PATCH until the target version is promotable.
func (r *DeploymentResource) patchDeployment(ctx context.Context, path string, body map[string]any, result *map[string]any) error {
	if _, changingDefaultVersion := body["default_version"]; !changingDefaultVersion {
		return r.client.Patch(ctx, path, body, result)
	}

	deadline := time.Now().Add(defaultVersionUnavailableRetryTimeout)
	for {
		err := r.client.Patch(ctx, path, body, result)
		if err == nil {
			return nil
		}

		var apiErr *client.UbiOpsError
		if !errors.As(err, &apiErr) || !isDefaultVersionNotReadyYet(apiErr) {
			return err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("%w (target version did not become ready within %s)", err, defaultVersionUnavailableRetryTimeout)
		}
		tflog.Info(ctx, "target version not yet available for promotion, retrying", map[string]any{"path": path})
		select {
		case <-time.After(defaultVersionUnavailableRetryInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// isDefaultVersionNotReadyYet matches transient "not built yet"/"doesn't exist yet" errors.
func isDefaultVersionNotReadyYet(apiErr *client.UbiOpsError) bool {
	switch {
	case apiErr.StatusCode == http.StatusBadRequest && apiErr.Message == "Default cannot be set to an unavailable version":
		// The version exists but its build/revision hasn't finished yet.
		return true
	case apiErr.StatusCode == http.StatusNotFound && apiErr.Message == "Deployment version was not found":
		// create hasn't landed yet - a race with the concurrent version create.
		return true
	default:
		return false
	}
}

func (r *DeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, fmt.Sprintf("/projects/%s/deployments/%s", data.ProjectName.ValueString(), data.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting deployment", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted deployment", map[string]any{"name": data.Name.ValueString()})
}

func (r *DeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID2(ctx, req.ID, "name", resp)
}

// fieldsToAPI converts a Terraform list of field objects to the API representation.
func fieldsToAPI(ctx context.Context, fieldsList types.List, diags *diag.Diagnostics) []map[string]string {
	var fields []struct {
		Name     string `tfsdk:"name"`
		DataType string `tfsdk:"data_type"`
	}

	diags.Append(fieldsList.ElementsAs(ctx, &fields, false)...)
	if diags.HasError() {
		return nil
	}

	result := make([]map[string]string, 0, len(fields))
	for _, f := range fields {
		result = append(result, map[string]string{
			"name":      f.Name,
			"data_type": f.DataType,
		})
	}
	return result
}

// readDeploymentResult maps the API response to the Terraform model.
func readDeploymentResult(_ context.Context, result map[string]any, data *DeploymentResourceModel) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["name"].(string); ok {
		data.Name = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["input_type"].(string); ok {
		data.InputType = types.StringValue(v)
	}
	if v, ok := result["output_type"].(string); ok {
		data.OutputType = types.StringValue(v)
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

	// Default version can be null.
	if v, ok := result["default_version"]; ok && v != nil {
		if s, ok := v.(string); ok {
			data.DefaultVersion = types.StringValue(s)
		}
	} else {
		data.DefaultVersion = types.StringNull()
	}

	// Input fields.
	data.InputFields = fieldsFromAPI(result["input_fields"])

	// Output fields.
	data.OutputFields = fieldsFromAPI(result["output_fields"])

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
			m, _ := types.MapValueFrom(context.Background(), types.StringType, vals)
			data.Labels = m
		} else {
			data.Labels = types.MapNull(types.StringType)
		}
	} else {
		data.Labels = types.MapNull(types.StringType)
	}
}

// fieldsFromAPI converts an API field list to a Terraform types.List.
func fieldsFromAPI(raw any) types.List {
	if raw == nil {
		return types.ListNull(fieldObjectType)
	}

	fields, ok := raw.([]any)
	if !ok {
		return types.ListNull(fieldObjectType)
	}

	if len(fields) == 0 {
		list, _ := types.ListValueFrom(context.Background(), fieldObjectType, []attr.Value{})
		return list
	}

	vals := make([]attr.Value, 0, len(fields))
	for _, f := range fields {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}
		name, _ := fm["name"].(string)
		dataType, _ := fm["data_type"].(string)

		obj, _ := types.ObjectValue(
			fieldObjectType.AttrTypes,
			map[string]attr.Value{
				"name":      types.StringValue(name),
				"data_type": types.StringValue(dataType),
			},
		)
		vals = append(vals, obj)
	}

	list, _ := types.ListValue(fieldObjectType, vals)
	return list
}
