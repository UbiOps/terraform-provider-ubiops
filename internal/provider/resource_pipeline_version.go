// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &PipelineVersionResource{}
	_ resource.ResourceWithImportState = &PipelineVersionResource{}
	_ resource.ResourceWithConfigure   = &PipelineVersionResource{}
)

// ── object types for the pipeline topology ────────────────────────────────────

// pipelineObjectType is the attr.Type for a single step in the pipeline graph.
var pipelineObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":               types.StringType,
		"reference_type":     types.StringType,
		"reference_name":     types.StringType,
		"version":            types.StringType,
		"configuration_json": types.StringType,
	},
}

// fieldMappingType is the attr.Type for a single field-to-field mapping.
var fieldMappingType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"source_field_name":      types.StringType,
		"destination_field_name": types.StringType,
	},
}

// attachmentSourceType is the attr.Type for one source within an attachment.
var attachmentSourceType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"source_name": types.StringType,
		"mapping":     types.ListType{ElemType: fieldMappingType},
	},
}

// pipelineAttachmentType is the attr.Type for a single attachment (edge) in the graph.
var pipelineAttachmentType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"destination_name": types.StringType,
		"sources":          types.ListType{ElemType: attachmentSourceType},
	},
}

// ── resource ──────────────────────────────────────────────────────────────────

// NewPipelineVersionResource returns a new pipeline version resource.
func NewPipelineVersionResource() resource.Resource {
	return &PipelineVersionResource{}
}

// PipelineVersionResource manages a UbiOps pipeline version.
type PipelineVersionResource struct {
	client *client.UbiOpsClient
}

// PipelineVersionResourceModel maps the pipeline version schema to Go types.
type PipelineVersionResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProjectName          types.String `tfsdk:"project_name"`
	PipelineName         types.String `tfsdk:"pipeline_name"`
	Version              types.String `tfsdk:"version"`
	Description          types.String `tfsdk:"description"`
	Labels               types.Map    `tfsdk:"labels"`
	RequestRetentionMode types.String `tfsdk:"request_retention_mode"`
	RequestRetentionTime types.Int64  `tfsdk:"request_retention_time"`
	Objects              types.List   `tfsdk:"objects"`
	Attachments          types.List   `tfsdk:"attachments"`
	CreationDate         types.String `tfsdk:"creation_date"`
	LastUpdated          types.String `tfsdk:"last_updated"`
}

func (r *PipelineVersionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_version"
}

func (r *PipelineVersionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UbiOps pipeline version.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the pipeline version (UUID)",
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
			"pipeline_name": schema.StringAttribute{
				MarkdownDescription: "The name of the pipeline.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Name of the version of the pipeline.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the pipeline version.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Key/value labels attached to this pipeline version.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"request_retention_mode": schema.StringAttribute{
				MarkdownDescription: "One of `none`, `metadata`, or `full`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("full"),
				Validators: []validator.String{
					stringvalidator.OneOf("none", "metadata", "full"),
				},
			},
			"request_retention_time": schema.Int64Attribute{
				MarkdownDescription: "Seconds to retain requests.",
				Optional:            true,
				Computed:            true,
			},

			// ── pipeline topology ──────────────────────────────────────────

			"objects": schema.ListNestedAttribute{
				MarkdownDescription: "Steps (deployments, pipelines, or operators) that form the pipeline graph.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Unique name for this step within the pipeline version.",
							Required:            true,
						},
						"reference_type": schema.StringAttribute{
							MarkdownDescription: "Type of the referenced object: `deployment`, `pipeline`, or `operator`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("deployment", "pipeline", "operator"),
							},
						},
						"reference_name": schema.StringAttribute{
							MarkdownDescription: "Name of the deployment, pipeline, or operator to reference.",
							Required:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Version of the referenced deployment or pipeline. Omit to use the default version.",
							Optional:            true,
							Computed:            true,
						},
						"configuration_json": schema.StringAttribute{
							MarkdownDescription: "JSON-encoded configuration for this object. For operators: `expression`, `input_fields`, `output_fields`, etc. For deployments/pipelines: `on_error` policy (`stop` or `continue`). Omit if not needed.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},

			"attachments": schema.ListNestedAttribute{
				MarkdownDescription: "Edges in the pipeline graph — wires outputs of one step to inputs of another.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"destination_name": schema.StringAttribute{
							MarkdownDescription: "Name of the destination step (or `pipeline_end`).",
							Required:            true,
						},
						"sources": schema.ListNestedAttribute{
							MarkdownDescription: "One or more source steps feeding this destination.",
							Required:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"source_name": schema.StringAttribute{
										MarkdownDescription: "Name of the source step (or `pipeline_start`).",
										Required:            true,
									},
									"mapping": schema.ListNestedAttribute{
										MarkdownDescription: "Field-level mapping from source outputs to destination inputs.",
										Optional:            true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"source_field_name": schema.StringAttribute{
													MarkdownDescription: "Output field on the source step.",
													Required:            true,
												},
												"destination_field_name": schema.StringAttribute{
													MarkdownDescription: "Input field on the destination step.",
													Required:            true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},

			"creation_date": schema.StringAttribute{
				MarkdownDescription: "The date when the pipeline version was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The date when the pipeline version was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *PipelineVersionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *PipelineVersionResource) basePath(projectName, pipelineName string) string {
	return fmt.Sprintf("/projects/%s/pipelines/%s/versions", projectName, pipelineName)
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

func (r *PipelineVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PipelineVersionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"version": data.Version.ValueString(),
	}

	setOptionalString(body, "description", data.Description)
	setOptionalString(body, "request_retention_mode", data.RequestRetentionMode)
	setOptionalInt64(body, "request_retention_time", data.RequestRetentionTime)

	if !data.Labels.IsNull() {
		labels := make(map[string]string)
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["labels"] = labels
	}

	if !data.Objects.IsNull() && !data.Objects.IsUnknown() {
		body["objects"] = pipelineObjectsToAPI(ctx, data.Objects, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.Attachments.IsNull() && !data.Attachments.IsUnknown() {
		body["attachments"] = pipelineAttachmentsToAPI(ctx, data.Attachments, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.PipelineName.ValueString())

	var result map[string]any
	if err := r.client.Post(ctx, bp, body, &result); err != nil {
		resp.Diagnostics.AddError("Error creating pipeline version", err.Error())
		return
	}

	readPipelineVersionResult(ctx, result, &data, &resp.Diagnostics)
	tflog.Trace(ctx, "created pipeline version", map[string]any{"version": data.Version.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PipelineVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.PipelineName.ValueString())

	var result map[string]any
	if err := r.client.Get(ctx, fmt.Sprintf("%s/%s", bp, data.Version.ValueString()), &result); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading pipeline version", err.Error())
		return
	}

	readPipelineVersionResult(ctx, result, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PipelineVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PipelineVersionResourceModel
	var state PipelineVersionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	setChangedString(body, "version", plan.Version, state.Version)
	setChangedString(body, "description", plan.Description, state.Description)
	setChangedString(body, "request_retention_mode", plan.RequestRetentionMode, state.RequestRetentionMode)

	if !plan.RequestRetentionTime.Equal(state.RequestRetentionTime) {
		if plan.RequestRetentionTime.IsNull() {
			body["request_retention_time"] = nil
		} else {
			body["request_retention_time"] = plan.RequestRetentionTime.ValueInt64()
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

	if !plan.Objects.Equal(state.Objects) {
		if plan.Objects.IsNull() {
			body["objects"] = []any{}
		} else {
			body["objects"] = pipelineObjectsToAPI(ctx, plan.Objects, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	if !plan.Attachments.Equal(state.Attachments) {
		if plan.Attachments.IsNull() {
			body["attachments"] = []any{}
		} else {
			body["attachments"] = pipelineAttachmentsToAPI(ctx, plan.Attachments, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	bp := r.basePath(state.ProjectName.ValueString(), state.PipelineName.ValueString())

	var result map[string]any
	if err := r.client.Patch(ctx, fmt.Sprintf("%s/%s", bp, state.Version.ValueString()), body, &result); err != nil {
		resp.Diagnostics.AddError("Error updating pipeline version", err.Error())
		return
	}

	readPipelineVersionResult(ctx, result, &plan, &resp.Diagnostics)
	tflog.Trace(ctx, "updated pipeline version", map[string]any{"version": plan.Version.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PipelineVersionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp := r.basePath(data.ProjectName.ValueString(), data.PipelineName.ValueString())
	if err := r.client.Delete(ctx, fmt.Sprintf("%s/%s", bp, data.Version.ValueString())); err != nil {
		resp.Diagnostics.AddError("Error deleting pipeline version", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted pipeline version", map[string]any{"version": data.Version.ValueString()})
}

func (r *PipelineVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID3(ctx, req.ID, "pipeline_name", "version", resp)
}

// ── API → Terraform ───────────────────────────────────────────────────────────

// readPipelineVersionResult maps the API response to the Terraform model.
// Only the fields the user controls are extracted — API-only fields are discarded.
func readPipelineVersionResult(ctx context.Context, result map[string]any, data *PipelineVersionResourceModel, diags *diag.Diagnostics) {
	if v, ok := result["id"].(string); ok {
		data.ID = types.StringValue(v)
	}
	if v, ok := result["version"].(string); ok {
		data.Version = types.StringValue(v)
	}
	if v, ok := result["description"].(string); ok {
		data.Description = types.StringValue(v)
	}
	if v, ok := result["request_retention_mode"].(string); ok {
		data.RequestRetentionMode = types.StringValue(v)
	}
	if v, ok := result["creation_date"].(string); ok {
		data.CreationDate = types.StringValue(v)
	}
	if v, ok := result["last_updated"].(string); ok {
		data.LastUpdated = types.StringValue(v)
	}

	readInt64Field(result, "request_retention_time", &data.RequestRetentionTime)

	// Labels.
	if v, ok := result["labels"]; ok && v != nil {
		if labelsMap, ok := v.(map[string]any); ok && len(labelsMap) > 0 {
			vals := make(map[string]string, len(labelsMap))
			for k, val := range labelsMap {
				if s, ok := val.(string); ok {
					vals[k] = s
				}
			}
			m, d := types.MapValueFrom(ctx, types.StringType, vals)
			diags.Append(d...)
			data.Labels = m
		} else {
			data.Labels = types.MapNull(types.StringType)
		}
	} else {
		data.Labels = types.MapNull(types.StringType)
	}

	// Objects.
	data.Objects = pipelineObjectsFromAPI(result["objects"])

	// Attachments.
	data.Attachments = pipelineAttachmentsFromAPI(result["attachments"])
}

// pipelineObjectsFromAPI converts the API objects list to a types.List,
// keeping only the fields the user controls (strips id, configuration, input_fields, etc.).
func pipelineObjectsFromAPI(raw any) types.List {
	null := types.ListNull(pipelineObjectType)
	if raw == nil {
		return null
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return null
	}

	vals := make([]attr.Value, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		refType, _ := m["reference_type"].(string)
		refName, _ := m["reference_name"].(string)

		var version attr.Value
		if v, ok := m["version"]; ok && v != nil {
			if s, ok := v.(string); ok && s != "" {
				version = types.StringValue(s)
			} else {
				version = types.StringNull()
			}
		} else {
			version = types.StringNull()
		}

		var configJSON attr.Value
		if cfg, ok := m["configuration"]; ok && cfg != nil {
			if cfgMap, ok := cfg.(map[string]any); ok && len(cfgMap) > 0 {
				b, err := json.Marshal(cfgMap)
				if err == nil {
					configJSON = types.StringValue(string(b))
				} else {
					configJSON = types.StringNull()
				}
			} else {
				configJSON = types.StringNull()
			}
		} else {
			configJSON = types.StringNull()
		}

		obj, _ := types.ObjectValue(pipelineObjectType.AttrTypes, map[string]attr.Value{
			"name":               types.StringValue(name),
			"reference_type":     types.StringValue(refType),
			"reference_name":     types.StringValue(refName),
			"version":            version,
			"configuration_json": configJSON,
		})
		vals = append(vals, obj)
	}

	v, _ := types.ListValue(pipelineObjectType, vals)
	return v
}

// pipelineAttachmentsFromAPI converts the API attachments list to a types.List,
// keeping only destination_name, source_name, and field mappings.
func pipelineAttachmentsFromAPI(raw any) types.List {
	null := types.ListNull(pipelineAttachmentType)
	if raw == nil {
		return null
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return null
	}

	// Sort by destination_name for deterministic state ordering. The API may
	// return attachments in a different order than the user specified; without
	// sorting, the state differs from the plan and Terraform errors with
	// "Provider produced inconsistent result after apply".
	sort.Slice(items, func(i, j int) bool {
		mi, _ := items[i].(map[string]any)
		mj, _ := items[j].(map[string]any)
		di, _ := mi["destination_name"].(string)
		dj, _ := mj["destination_name"].(string)
		return di < dj
	})

	attachVals := make([]attr.Value, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		destName, _ := m["destination_name"].(string)

		// sources
		var sourceVals []attr.Value
		if rawSources, ok := m["sources"].([]any); ok {
			for _, rs := range rawSources {
				sm, ok := rs.(map[string]any)
				if !ok {
					continue
				}
				sourceName, _ := sm["source_name"].(string)

				// mapping
				var mappingVals []attr.Value
				if rawMapping, ok := sm["mapping"].([]any); ok {
					for _, rm := range rawMapping {
						mm, ok := rm.(map[string]any)
						if !ok {
							continue
						}
						src, _ := mm["source_field_name"].(string)
						dst, _ := mm["destination_field_name"].(string)
						obj, _ := types.ObjectValue(fieldMappingType.AttrTypes, map[string]attr.Value{
							"source_field_name":      types.StringValue(src),
							"destination_field_name": types.StringValue(dst),
						})
						mappingVals = append(mappingVals, obj)
					}
				}
				mappingList, _ := types.ListValue(fieldMappingType, mappingVals)

				sourceObj, _ := types.ObjectValue(attachmentSourceType.AttrTypes, map[string]attr.Value{
					"source_name": types.StringValue(sourceName),
					"mapping":     mappingList,
				})
				sourceVals = append(sourceVals, sourceObj)
			}
		}
		sourcesList, _ := types.ListValue(attachmentSourceType, sourceVals)

		attachObj, _ := types.ObjectValue(pipelineAttachmentType.AttrTypes, map[string]attr.Value{
			"destination_name": types.StringValue(destName),
			"sources":          sourcesList,
		})
		attachVals = append(attachVals, attachObj)
	}

	v, _ := types.ListValue(pipelineAttachmentType, attachVals)
	return v
}

// ── Terraform → API ───────────────────────────────────────────────────────────

func pipelineObjectsToAPI(ctx context.Context, list types.List, diags *diag.Diagnostics) []map[string]any {
	var items []struct {
		Name              string       `tfsdk:"name"`
		ReferenceType     string       `tfsdk:"reference_type"`
		ReferenceName     string       `tfsdk:"reference_name"`
		Version           types.String `tfsdk:"version"`
		ConfigurationJSON types.String `tfsdk:"configuration_json"`
	}
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if diags.HasError() {
		return nil
	}

	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		obj := map[string]any{
			"name":           it.Name,
			"reference_type": it.ReferenceType,
			"reference_name": it.ReferenceName,
		}
		if !it.Version.IsNull() && !it.Version.IsUnknown() && it.Version.ValueString() != "" {
			obj["version"] = it.Version.ValueString()
		}
		if !it.ConfigurationJSON.IsNull() && !it.ConfigurationJSON.IsUnknown() {
			var cfg map[string]any
			if err := json.Unmarshal([]byte(it.ConfigurationJSON.ValueString()), &cfg); err == nil {
				obj["configuration"] = cfg
			}
		}
		out = append(out, obj)
	}
	return out
}

func pipelineAttachmentsToAPI(ctx context.Context, list types.List, diags *diag.Diagnostics) []map[string]any {
	type mappingItem struct {
		SourceFieldName      string `tfsdk:"source_field_name"`
		DestinationFieldName string `tfsdk:"destination_field_name"`
	}
	type sourceItem struct {
		SourceName string        `tfsdk:"source_name"`
		Mapping    []mappingItem `tfsdk:"mapping"`
	}
	type attachItem struct {
		DestinationName string       `tfsdk:"destination_name"`
		Sources         []sourceItem `tfsdk:"sources"`
	}

	var items []attachItem
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if diags.HasError() {
		return nil
	}

	out := make([]map[string]any, 0, len(items))
	for _, att := range items {
		sources := make([]map[string]any, 0, len(att.Sources))
		for _, src := range att.Sources {
			mapping := make([]map[string]string, 0, len(src.Mapping))
			for _, mp := range src.Mapping {
				mapping = append(mapping, map[string]string{
					"source_field_name":      mp.SourceFieldName,
					"destination_field_name": mp.DestinationFieldName,
				})
			}
			sources = append(sources, map[string]any{
				"source_name": src.SourceName,
				"mapping":     mapping,
			})
		}
		out = append(out, map[string]any{
			"destination_name": att.DestinationName,
			"sources":          sources,
		})
	}
	return out
}
