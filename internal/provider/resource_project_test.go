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

func TestAccProjectResource(t *testing.T) {
	orgName := os.Getenv("UBIOPS_ORGANIZATION")
	if orgName == "" {
		t.Skip("UBIOPS_ORGANIZATION must be set for acceptance tests")
	}

	projectName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read.
			{
				Config: testAccProjectResourceConfig(projectName, orgName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_project.test",
						tfjsonpath.New("organization_name"),
						knownvalue.StringExact(orgName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_project.test",
						tfjsonpath.New("advanced_permissions"),
						knownvalue.Bool(false),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name.
			{
				Config: testAccProjectResourceConfig(projectName+"-updated", orgName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName+"-updated"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccProjectResourceConfig(name, orgName string) string {
	return fmt.Sprintf(`
resource "ubiops_project" "test" {
  name              = %[1]q
  organization_name = %[2]q
}
`, name, orgName)
}
