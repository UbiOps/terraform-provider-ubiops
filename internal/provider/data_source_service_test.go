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

func TestAccServiceDataSource(t *testing.T) {
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
			{
				Config: testAccServiceDataSourceConfig(projectName, deploymentName, serviceName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.ubiops_service.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(serviceName),
					),
					statecheck.ExpectKnownValue(
						"data.ubiops_service.test",
						tfjsonpath.New("deployment"),
						knownvalue.StringExact(deploymentName),
					),
					statecheck.ExpectKnownValue(
						"data.ubiops_service.test",
						tfjsonpath.New("port"),
						knownvalue.Int64Exact(8080),
					),
				},
			},
		},
	})
}

func testAccServiceDataSourceConfig(projectName, deploymentName, serviceName string) string {
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

  depends_on = [ubiops_deployment_version.test]
}

data "ubiops_service" "test" {
  project_name = %[1]q
  name         = ubiops_service.test.name
}
`, projectName, deploymentName, serviceName)
}
