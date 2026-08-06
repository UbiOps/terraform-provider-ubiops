// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPipelineVersionResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	pipelineName := fmt.Sprintf("tf-acc-test-%s", t.Name())
	versionName := "v1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccPipelineVersionResourceConfig(projectName, pipelineName, versionName, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact(versionName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("request_retention_mode"),
						knownvalue.StringExact("full"),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_pipeline_version.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s/%s", projectName, pipelineName, versionName),
				ImportStateVerify: true,
			},
			// Update description.
			{
				Config: testAccPipelineVersionResourceConfig(projectName, pipelineName, versionName, "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccPipelineVersionResourceConfig(projectName, pipelineName, version, description string) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("\n  description = %q", description)
	}
	return fmt.Sprintf(`
resource "ubiops_pipeline" "test" {
  project_name = %[1]q
  name         = %[2]q
  input_type   = "structured"
  output_type  = "structured"

  input_fields {
    name      = "input"
    data_type = "string"
  }

  output_fields {
    name      = "output"
    data_type = "string"
  }
}

resource "ubiops_pipeline_version" "test" {
  project_name = %[1]q
  pipeline_name = ubiops_pipeline.test.name
  version       = %[3]q%[4]s
}
`, projectName, pipelineName, version, desc)
}
