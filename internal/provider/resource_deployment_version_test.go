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

func TestAccDeploymentVersionResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)
	versionName := "v1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_deployment_version", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/deployments/%s/versions/%s", a["project_name"], a["deployment_name"], a["version"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccDeploymentVersionResourceConfig(projectName, deploymentName, versionName, 0, 5),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact(versionName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("minimum_instances"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("maximum_instances"),
						knownvalue.Int64Exact(5),
					),
				},
			},
			// ImportState.
			{
				ResourceName:            "ubiops_deployment_version.test",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s/%s", projectName, deploymentName, versionName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"build_timeout", "static_ip"},
			},
			// Update scaling.
			{
				Config: testAccDeploymentVersionResourceConfig(projectName, deploymentName, versionName, 1, 3),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("minimum_instances"),
						knownvalue.Int64Exact(1),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("maximum_instances"),
						knownvalue.Int64Exact(3),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccDeploymentVersionResourceConfig(projectName, deploymentName, version string, minInstances, maxInstances int) string {
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
  project_name      = %[1]q
  deployment_name   = ubiops_deployment.test.name
  version           = %[3]q
  minimum_instances = %[4]d
  maximum_instances = %[5]d
}
`, projectName, deploymentName, version, minInstances, maxInstances)
}
