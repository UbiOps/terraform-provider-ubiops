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

func TestAccServiceResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t) + "-dep"
	serviceName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccServiceResourceConfig(projectName, deploymentName, serviceName, "A test service"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_service.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(serviceName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_service.test",
						tfjsonpath.New("port"),
						knownvalue.Int64Exact(8080),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_service.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectName, serviceName),
				ImportStateVerify: true,
			},
			// Update description.
			{
				Config: testAccServiceResourceConfig(projectName, deploymentName, serviceName, "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_service.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccServiceResourceConfig(projectName, deploymentName, serviceName, description string) string {
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

resource "ubiops_service" "test" {
  project_name = %[1]q
  name         = %[3]q
  deployment   = ubiops_deployment.test.name
  port         = 8080
  description  = %[4]q

  depends_on = [ubiops_deployment_version.test]
}
`, projectName, deploymentName, serviceName, description)
}
