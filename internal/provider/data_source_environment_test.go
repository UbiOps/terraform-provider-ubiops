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

func TestAccEnvironmentDataSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	envName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSourceConfig(projectName, envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.ubiops_environment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envName),
					),
					statecheck.ExpectKnownValue(
						"data.ubiops_environment.test",
						tfjsonpath.New("base_environment"),
						knownvalue.StringExact("python3-12"),
					),
				},
			},
		},
	})
}

func testAccEnvironmentDataSourceConfig(projectName, envName string) string {
	return fmt.Sprintf(`
resource "ubiops_environment" "test" {
  project_name     = %[1]q
  name             = %[2]q
  display_name     = %[2]q
  base_environment = "python3-12"
}

data "ubiops_environment" "test" {
  project_name = %[1]q
  name         = ubiops_environment.test.name
}
`, projectName, envName)
}
