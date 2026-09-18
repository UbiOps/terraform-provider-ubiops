// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &FileResource{}
	_ resource.ResourceWithImportState = &FileResource{}
	_ resource.ResourceWithConfigure   = &FileResource{}
)

// NewFileResource returns a new file resource.
func NewFileResource() resource.Resource {
	return &FileResource{}
}

// FileResource manages a UbiOps file in a bucket.
type FileResource struct {
	client *client.UbiOpsClient
}

// FileResourceModel maps the file schema to Go types.
type FileResourceModel struct {
	ProjectName types.String `tfsdk:"project_name"`
	BucketName  types.String `tfsdk:"bucket_name"`
	File        types.String `tfsdk:"file"`
	Size        types.Int64  `tfsdk:"size"`
	TimeCreated types.String `tfsdk:"time_created"`
}

func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a file in a UbiOps bucket. Creates an empty file placeholder.",

		Attributes: map[string]schema.Attribute{
			"project_name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bucket_name": schema.StringAttribute{
				MarkdownDescription: "The name of the bucket.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file": schema.StringAttribute{
				MarkdownDescription: "Name of the file",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "Size of the file in bytes",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"time_created": schema.StringAttribute{
				MarkdownDescription: "The time that the file was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureClient(req.ProviderData, &r.client, &resp.Diagnostics)
}

func (r *FileResource) filePath(projectName, bucketName, file string) string {
	return fmt.Sprintf("/projects/%s/buckets/%s/files/%s", projectName, bucketName, url.PathEscape(file))
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fp := r.filePath(data.ProjectName.ValueString(), data.BucketName.ValueString(), data.File.ValueString())

	// Step 1: request a pre-signed upload URL from the API.
	var uploadResp map[string]any
	if err := r.client.Post(ctx, fp, map[string]any{}, &uploadResp); err != nil {
		resp.Diagnostics.AddError("Error creating file", err.Error())
		return
	}

	// Step 2: the UbiOps file API is a two-step process - POST returns a signed
	// upload URL; PUT (even zero bytes) actually stores the object.
	if uploadURL, ok := uploadResp["url"].(string); ok && uploadURL != "" {
		putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader([]byte{}))
		if err != nil {
			resp.Diagnostics.AddError("Error building file upload request", err.Error())
			return
		}
		putResp, err := r.client.HTTPClient.Do(putReq)
		if err != nil {
			resp.Diagnostics.AddError("Error uploading file", err.Error())
			return
		}
		putResp.Body.Close()
		if putResp.StatusCode >= 400 {
			resp.Diagnostics.AddError("Error uploading file", fmt.Sprintf("upload returned HTTP %d", putResp.StatusCode))
			return
		}
	}

	// Step 3: read the file metadata now that the object exists.
	var result map[string]any
	if err := r.client.Get(ctx, fp, &result); err != nil {
		resp.Diagnostics.AddError("Error reading file after create", err.Error())
		return
	}

	readFileResult(result, &data)

	tflog.Trace(ctx, "created file", map[string]any{"file": data.File.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fp := r.filePath(data.ProjectName.ValueString(), data.BucketName.ValueString(), data.File.ValueString())

	var result map[string]any
	err := r.client.Get(ctx, fp, &result)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading file", err.Error())
		return
	}

	readFileResult(result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes require replace, so Update is never called.
	resp.Diagnostics.AddError("Unexpected update", "Files cannot be updated in-place.")
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fp := r.filePath(data.ProjectName.ValueString(), data.BucketName.ValueString(), data.File.ValueString())

	err := r.client.Delete(ctx, fp)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting file", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted file", map[string]any{"file": data.File.ValueString()})
}

func (r *FileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parseImportID3(ctx, req.ID, "bucket_name", "file", resp)
}

// readFileResult maps the API response to the Terraform model.
func readFileResult(result map[string]any, data *FileResourceModel) {
	if v, ok := result["file"].(string); ok {
		data.File = types.StringValue(v)
	}
	if v, ok := result["time_created"].(string); ok {
		data.TimeCreated = types.StringValue(v)
	} else {
		data.TimeCreated = types.StringNull()
	}
	// size may be absent on a freshly created empty file.
	if v, ok := result["size"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			data.Size = types.Int64Value(int64(f))
		} else {
			data.Size = types.Int64Null()
		}
	} else {
		data.Size = types.Int64Null()
	}
}
