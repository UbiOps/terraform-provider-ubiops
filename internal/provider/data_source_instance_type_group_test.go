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

func TestAccInstanceTypeGroupDataSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	groupName := fmt.Sprintf("tf-acc-test-%s", t.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceTypeGroupDataSourceConfig(projectName, groupName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.ubiops_instance_type_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(groupName),
					),
				},
			},
		},
	})
}

func testAccInstanceTypeGroupDataSourceConfig(projectName, groupName string) string {
	return fmt.Sprintf(`
resource "ubiops_instance_type_group" "test" {
  project_name = %[1]q
  name         = %[2]q
}

data "ubiops_instance_type_group" "test" {
  project_name = %[1]q
  id           = ubiops_instance_type_group.test.id
}
`, projectName, groupName)
}
