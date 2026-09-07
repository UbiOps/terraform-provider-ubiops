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

func TestAccDeploymentVersionEnvironmentVariableResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)
	versionName := "v1"
	envVarName := "TF_ACC_TEST_VAR"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_deployment_version_environment_variable", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/deployments/%s/versions/%s/environment-variables/%s", a["project_name"], a["deployment_name"], a["version"], a["id"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccDeploymentVersionEnvVarResourceConfig(projectName, deploymentName, versionName, envVarName, "initial-value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version_environment_variable.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envVarName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version_environment_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("initial-value"),
					),
				},
			},
			// Update value.
			{
				Config: testAccDeploymentVersionEnvVarResourceConfig(projectName, deploymentName, versionName, envVarName, "updated-value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version_environment_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("updated-value"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccDeploymentVersionEnvVarResourceConfig(projectName, deploymentName, versionName, envVarName, value string) string {
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

resource "ubiops_deployment_version" "test" {
  project_name    = %[1]q
  deployment_name = ubiops_deployment.test.name
  version         = %[3]q
}

resource "ubiops_deployment_version_environment_variable" "test" {
  project_name    = %[1]q
  deployment_name = ubiops_deployment.test.name
  version         = ubiops_deployment_version.test.version
  name            = %[4]q
  value           = %[5]q
}
`, projectName, deploymentName, versionName, envVarName, value)
}
