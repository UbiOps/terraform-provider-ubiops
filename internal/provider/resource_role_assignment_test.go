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

func TestAccRoleAssignmentResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	suName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_role_assignment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/role-assignments/%s", a["project_name"], a["id"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccRoleAssignmentResourceConfig(projectName, suName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_role_assignment.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("project-viewer"),
					),
					statecheck.ExpectKnownValue(
						"ubiops_role_assignment.test",
						tfjsonpath.New("assignee_type"),
						knownvalue.StringExact("user"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccRoleAssignmentResourceConfig(projectName, suName string) string {
	return fmt.Sprintf(`
resource "ubiops_service_user" "test" {
  project_name = %[1]q
  name         = %[2]q
}

resource "ubiops_role_assignment" "test" {
  project_name  = %[1]q
  role          = "project-viewer"
  assignee      = ubiops_service_user.test.id
  assignee_type = "user"
}
`, projectName, suName)
}
