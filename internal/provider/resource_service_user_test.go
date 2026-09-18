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

func TestAccServiceUserResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	suName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_service_user", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/service-users/%s", a["project_name"], a["id"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccServiceUserResourceConfig(projectName, suName, "Test service user"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_service_user.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(suName),
					),
				},
			},
			{
				ResourceName:      "ubiops_service_user.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_service_user.test", "project_name", "id"),
				ImportStateVerify: true,
			},
			// Update.
			{
				Config: testAccServiceUserResourceConfig(projectName, suName, "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_service_user.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccServiceUserResourceConfig(projectName, name, description string) string {
	return fmt.Sprintf(`
resource "ubiops_service_user" "test" {
  project_name = %[1]q
  name         = %[2]q
  description  = %[3]q
}
`, projectName, name, description)
}
