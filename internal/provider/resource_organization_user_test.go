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

func TestAccOrganizationUserResource(t *testing.T) {
	orgName := os.Getenv("UBIOPS_ORGANIZATION")
	if orgName == "" {
		t.Skip("UBIOPS_ORGANIZATION must be set for acceptance tests")
	}

	testEmail := os.Getenv("UBIOPS_TEST_USER_EMAIL")
	if testEmail == "" {
		t.Skip("UBIOPS_TEST_USER_EMAIL must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_organization_user", func(a map[string]string) string {
			return fmt.Sprintf("/organizations/%s/users/%s", a["organization_name"], a["id"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccOrganizationUserResourceConfig(orgName, testEmail, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_organization_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact(testEmail),
					),
					statecheck.ExpectKnownValue(
						"ubiops_organization_user.test",
						tfjsonpath.New("admin"),
						knownvalue.Bool(false),
					),
				},
			},
			// Update admin flag.
			{
				Config: testAccOrganizationUserResourceConfig(orgName, testEmail, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_organization_user.test",
						tfjsonpath.New("admin"),
						knownvalue.Bool(true),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccOrganizationUserResourceConfig(orgName, email string, admin bool) string {
	return fmt.Sprintf(`
resource "ubiops_organization_user" "test" {
  organization_name = %[1]q
  email             = %[2]q
  admin             = %[3]t
}
`, orgName, email, admin)
}
