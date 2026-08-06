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
)

// configureClient extracts the UbiOps client from provider data and sets it on the target pointer.
func configureClient(providerData any, target **client.UbiOpsClient, diags *diag.Diagnostics) {
	if providerData == nil {
		return
	}

	c, ok := providerData.(*client.UbiOpsClient)
	if !ok {
		diags.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *client.UbiOpsClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return
	}

	*target = c
}

// parseImportID2 parses a two-part import ID in the format "project_name/attr2".
func parseImportID2(ctx context.Context, id string, attr2 string, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: project_name/%s, got: %s", attr2, id),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr2), parts[1])...)
}

// parseImportID3 parses a three-part import ID in the format "project_name/attr2/attr3".
func parseImportID3(ctx context.Context, id string, attr2, attr3 string, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: project_name/%s/%s, got: %s", attr2, attr3, id),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr2), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr3), parts[2])...)
}
