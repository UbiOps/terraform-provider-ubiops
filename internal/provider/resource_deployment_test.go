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

func TestAccDeploymentResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := fmt.Sprintf("tf-acc-test-%s", t.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with structured input/output.
			{
				Config: testAccDeploymentResourceConfig(projectName, deploymentName, "structured"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(deploymentName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("input_type"),
						knownvalue.StringExact("structured"),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("project_name"),
						knownvalue.StringExact(projectName),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_deployment.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectName, deploymentName),
				ImportStateVerify: true,
			},
			// Update description.
			{
				Config: testAccDeploymentResourceConfigUpdated(projectName, deploymentName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccDeploymentResourceConfig(projectName, name, inputType string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name = %[1]q
  name         = %[2]q
  input_type   = %[3]q
  output_type  = %[3]q

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
`, projectName, name, inputType)
}

func testAccDeploymentResourceConfigUpdated(projectName, name string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name = %[1]q
  name         = %[2]q
  description  = "Updated description"
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
`, projectName, name)
}
