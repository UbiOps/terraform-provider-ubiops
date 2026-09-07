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

func TestAccFileResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	bucketName := testAccResourceName(t)
	fileName := "test-file.txt"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_file", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/buckets/%s/files/%s", a["project_name"], a["bucket_name"], a["file"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccFileResourceConfig(projectName, bucketName, fileName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_file.test",
						tfjsonpath.New("file"),
						knownvalue.StringExact(fileName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_file.test",
						tfjsonpath.New("bucket_name"),
						knownvalue.StringExact(bucketName),
					),
				},
			},
			// ImportState.
			{
				ResourceName:                         "ubiops_file.test",
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("%s/%s/%s", projectName, bucketName, fileName),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "file",
			},
			// Delete is automatic.
		},
	})
}

func testAccFileResourceConfig(projectName, bucketName, fileName string) string {
	return fmt.Sprintf(`
resource "ubiops_bucket" "test" {
  project_name = %[1]q
  name         = %[2]q
}

resource "ubiops_file" "test" {
  project_name = %[1]q
  bucket_name  = ubiops_bucket.test.name
  file         = %[3]q
}
`, projectName, bucketName, fileName)
}
