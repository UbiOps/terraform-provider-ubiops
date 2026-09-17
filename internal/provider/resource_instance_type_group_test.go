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

func TestAccInstanceTypeGroupResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	groupName := testAccResourceName(t)
	instanceTypeID := testAccFirstInstanceTypeID(t, projectName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_instance_type_group", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/instance-type-groups/%s", a["project_name"], a["id"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccInstanceTypeGroupResourceConfig(projectName, groupName, instanceTypeID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_instance_type_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(groupName),
					),
				},
			},
			{
				ResourceName:      "ubiops_instance_type_group.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_instance_type_group.test", "project_name", "id"),
				ImportStateVerify: true,
			},
			// Update name.
			{
				Config: testAccInstanceTypeGroupResourceConfig(projectName, groupName+"-updated", instanceTypeID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_instance_type_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(groupName+"-updated"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccInstanceTypeGroupResourceConfig(projectName, name, instanceTypeID string) string {
	return fmt.Sprintf(`
resource "ubiops_instance_type_group" "test" {
  project_name = %[1]q
  name         = %[2]q

  instance_types_json = jsonencode([
    {
      id       = %[3]q
      priority = 0
    }
  ])
}
`, projectName, name, instanceTypeID)
}
