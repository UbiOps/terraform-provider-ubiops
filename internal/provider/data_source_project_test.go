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

func TestAccProjectDataSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDataSourceConfig(projectName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.ubiops_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(projectName),
					),
				},
			},
		},
	})
}

func testAccProjectDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "ubiops_project" "test" {
  name = %[1]q
}
`, name)
}
