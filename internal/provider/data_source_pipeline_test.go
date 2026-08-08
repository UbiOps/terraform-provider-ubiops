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

func TestAccPipelineDataSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	pipelineName := fmt.Sprintf("tf-acc-test-%s", t.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineDataSourceConfig(projectName, pipelineName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.ubiops_pipeline.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(pipelineName),
					),
					statecheck.ExpectKnownValue(
						"data.ubiops_pipeline.test",
						tfjsonpath.New("input_type"),
						knownvalue.StringExact("structured"),
					),
				},
			},
		},
	})
}

func testAccPipelineDataSourceConfig(projectName, pipelineName string) string {
	return fmt.Sprintf(`
resource "ubiops_pipeline" "test" {
  project_name = %[1]q
  name         = %[2]q
  input_type   = "structured"
  output_type  = "structured"

  input_fields = [
    {
      name      = "input"
      data_type = "string"
    }
  ]

  output_fields = [
    {
      name      = "output"
      data_type = "string"
    }
  ]
}

data "ubiops_pipeline" "test" {
  project_name = %[1]q
  name         = ubiops_pipeline.test.name
}
`, projectName, pipelineName)
}
