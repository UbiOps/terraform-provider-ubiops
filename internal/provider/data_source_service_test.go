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

	deploymentName := fmt.Sprintf("tf-acc-test-%s-dep", t.Name())
	serviceName := fmt.Sprintf("tf-acc-test-%s", t.Name())

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

resource "ubiops_service" "test" {
  project_name = %[1]q
  name         = %[3]q
  deployment   = ubiops_deployment.test.name
  port         = 8080
}

data "ubiops_service" "test" {
  project_name = %[1]q
  name         = ubiops_service.test.name
}
`, projectName, deploymentName, serviceName)
}
