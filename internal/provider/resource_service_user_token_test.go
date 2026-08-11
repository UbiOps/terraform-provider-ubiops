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
)

func TestAccServiceUserTokenResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	suName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccServiceUserTokenResourceConfig(projectName, suName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue(
						"token_set",
						knownvalue.Bool(true),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccServiceUserTokenResourceConfig(projectName, suName string) string {
	return fmt.Sprintf(`
resource "ubiops_service_user" "test" {
  project_name = %[1]q
  name         = %[2]q
}

resource "ubiops_service_user_token" "test" {
  project_name    = %[1]q
  service_user_id = ubiops_service_user.test.id
}

output "token_set" {
  value     = ubiops_service_user_token.test.token != ""
  sensitive = true
}
`, projectName, suName)
}
