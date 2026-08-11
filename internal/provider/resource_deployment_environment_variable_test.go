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

func TestAccDeploymentEnvironmentVariableResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)
	envVarName := "TF_ACC_TEST_VAR"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccDeploymentEnvVarResourceConfig(projectName, deploymentName, envVarName, "initial-value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_environment_variable.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envVarName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment_environment_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("initial-value"),
					),
				},
			},
			// Update value.
			{
				Config: testAccDeploymentEnvVarResourceConfig(projectName, deploymentName, envVarName, "updated-value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_environment_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("updated-value"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccDeploymentEnvVarResourceConfig(projectName, deploymentName, envVarName, value string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
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

resource "ubiops_deployment_environment_variable" "test" {
  project_name    = %[1]q
  deployment_name = ubiops_deployment.test.name
  name            = %[3]q
  value           = %[4]q
}
`, projectName, deploymentName, envVarName, value)
}
