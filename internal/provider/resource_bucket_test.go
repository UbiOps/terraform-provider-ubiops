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

func TestAccBucketResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	bucketName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_bucket", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/buckets/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccBucketResourceConfig(projectName, bucketName, "A test bucket"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(bucketName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_bucket.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A test bucket"),
					),
				},
			},
			// ImportState.
			{
				ResourceName:            "ubiops_bucket.test",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s", projectName, bucketName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credentials"},
			},
			// Update description.
			{
				Config: testAccBucketResourceConfig(projectName, bucketName, "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_bucket.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccBucketResourceConfig(projectName, bucketName, description string) string {
	return fmt.Sprintf(`
resource "ubiops_bucket" "test" {
  project_name = %[1]q
  name         = %[2]q
  description  = %[3]q
}
`, projectName, bucketName, description)
}
