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

func TestAccRequestScheduleResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t) + "-dep"
	scheduleName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccRequestScheduleResourceConfig(projectName, deploymentName, scheduleName, "0 0 * * *"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_request_schedule.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(scheduleName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_request_schedule.test",
						tfjsonpath.New("schedule"),
						knownvalue.StringExact("0 0 * * *"),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_request_schedule.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectName, scheduleName),
				ImportStateVerify: true,
			},
			// Update schedule.
			{
				Config: testAccRequestScheduleResourceConfig(projectName, deploymentName, scheduleName, "0 12 * * *"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_request_schedule.test",
						tfjsonpath.New("schedule"),
						knownvalue.StringExact("0 12 * * *"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccRequestScheduleResourceConfig(projectName, deploymentName, scheduleName, schedule string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name    = %[1]q
  name            = %[2]q
  input_type      = "structured"
  output_type     = "structured"
  default_version = "v1"

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

resource "ubiops_deployment_version" "test" {
  project_name      = %[1]q
  deployment_name   = ubiops_deployment.test.name
  version           = "v1"
  minimum_instances = 0
  maximum_instances = 1
}

resource "ubiops_request_schedule" "test" {
  project_name = %[1]q
  name         = %[3]q
  object_type  = "deployment"
  object_name  = ubiops_deployment.test.name
  schedule     = %[4]q
  enabled      = false

  depends_on = [ubiops_deployment_version.test]
}
`, projectName, deploymentName, scheduleName, schedule)
}
