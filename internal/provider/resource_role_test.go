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

func TestAccRoleResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	roleName := fmt.Sprintf("tf-acc-test-%s", t.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with two read-only deployment permissions.
			{
				Config: testAccRoleResourceConfig(projectName, roleName, `["deployments.list", "deployments.get"]`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(roleName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_role.test",
						tfjsonpath.New("permissions"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_role.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectName, roleName),
				ImportStateVerify: true,
			},
			// Extend permissions — verifies the update path rewrites the full
			// permission list rather than appending or leaving stale entries.
			{
				Config: testAccRoleResourceConfig(projectName, roleName, `["deployments.list", "deployments.get", "deployments.versions.list", "deployments.versions.get"]`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_role.test",
						tfjsonpath.New("permissions"),
						knownvalue.ListSizeExact(4),
					),
				},
			},
			// Shrink permissions — verifies removed entries are not left on the API.
			{
				Config: testAccRoleResourceConfig(projectName, roleName, `["deployments.list"]`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_role.test",
						tfjsonpath.New("permissions"),
						knownvalue.ListSizeExact(1),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccRoleResourceConfig(projectName, roleName, permissions string) string {
	return fmt.Sprintf(`
resource "ubiops_role" "test" {
  project_name = %[1]q
  name         = %[2]q
  permissions  = %[3]s
}
`, projectName, roleName, permissions)
}
