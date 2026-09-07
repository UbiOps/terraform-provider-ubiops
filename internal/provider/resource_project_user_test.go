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

func TestAccProjectUserResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	testUserID := os.Getenv("UBIOPS_TEST_USER_ID")
	if testUserID == "" {
		t.Skip("UBIOPS_TEST_USER_ID must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_project_user", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/users/%s", a["project_name"], a["id"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccProjectUserResourceConfig(projectName, testUserID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_project_user.test",
						tfjsonpath.New("user_id"),
						knownvalue.StringExact(testUserID),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccProjectUserResourceConfig(projectName, userID string) string {
	return fmt.Sprintf(`
resource "ubiops_project_user" "test" {
  project_name = %[1]q
  user_id      = %[2]q
}
`, projectName, userID)
}
