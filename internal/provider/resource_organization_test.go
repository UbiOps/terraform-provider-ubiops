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

func TestAccOrganizationResource(t *testing.T) {
	// Organization management requires special permissions.
	orgName := os.Getenv("UBIOPS_TEST_ORGANIZATION")
	if orgName == "" {
		t.Skip("UBIOPS_TEST_ORGANIZATION must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccOrganizationResourceConfig(orgName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_organization.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(orgName),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_organization.test",
				ImportState:       true,
				ImportStateId:     orgName,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"subscription_end_date",
					"subscription_start_date",
				},
			},
			// Delete removes from state only (API does not support delete).
		},
	})
}

func testAccOrganizationResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "ubiops_organization" "test" {
  name = %[1]q
}
`, name)
}
