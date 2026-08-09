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

func TestAccEnvironmentResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	envName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccEnvironmentResourceConfig(projectName, envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("project_name"),
						knownvalue.StringExact(projectName),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_environment.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectName, envName),
				ImportStateVerify: true,
			},
			// Update description.
			{
				Config: testAccEnvironmentResourceConfigUpdated(projectName, envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccEnvironmentResourceConfig(projectName, name string) string {
	return fmt.Sprintf(`
resource "ubiops_environment" "test" {
  project_name     = %[1]q
  name             = %[2]q
  base_environment = "python3-12"
}
`, projectName, name)
}

func testAccEnvironmentResourceConfigUpdated(projectName, name string) string {
	return fmt.Sprintf(`
resource "ubiops_environment" "test" {
  project_name     = %[1]q
  name             = %[2]q
  base_environment = "python3-12"
  description      = "Updated description"
}
`, projectName, name)
}
