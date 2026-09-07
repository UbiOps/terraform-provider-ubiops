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

func TestAccProjectEnvironmentVariableResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	envVarName := fmt.Sprintf("TF_ACC_TEST_%s", t.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_project_environment_variable", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/environment-variables/%s", a["project_name"], a["id"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccProjectEnvVarResourceConfig(projectName, envVarName, "initial-value", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_project_environment_variable.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envVarName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_project_environment_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("initial-value"),
					),
				},
			},
			// Update value.
			{
				Config: testAccProjectEnvVarResourceConfig(projectName, envVarName, "updated-value", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_project_environment_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("updated-value"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccProjectEnvVarResourceConfig(projectName, name, value string, secret bool) string {
	return fmt.Sprintf(`
resource "ubiops_project_environment_variable" "test" {
  project_name = %[1]q
  name         = %[2]q
  value        = %[3]q
  secret       = %[4]t
}
`, projectName, name, value, secret)
}
