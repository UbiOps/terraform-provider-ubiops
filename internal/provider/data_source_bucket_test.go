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

func TestAccBucketDataSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	bucketName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketDataSourceConfig(projectName, bucketName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.ubiops_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(bucketName),
					),
					statecheck.ExpectKnownValue(
						"data.ubiops_bucket.test",
						tfjsonpath.New("bucket_provider"),
						knownvalue.StringExact("ubiops"),
					),
				},
			},
		},
	})
}

func testAccBucketDataSourceConfig(projectName, bucketName string) string {
	return fmt.Sprintf(`
resource "ubiops_bucket" "test" {
  project_name = %[1]q
  name         = %[2]q
}

data "ubiops_bucket" "test" {
  project_name = %[1]q
  name         = ubiops_bucket.test.name
}
`, projectName, bucketName)
}
